/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package instance

import (
	"fmt"
	"os"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
)

func Start(c geneos.Instance) (err error) {
	pid, err := GetPID(c)
	if err == nil {
		fmt.Printf("%s already running with PID %d\n", c, pid)
		return
	}

	if IsDisabled(c) {
		return geneos.ErrDisabled
	}

	binary := c.Config().GetString("program")
	if _, err = c.Host().Stat(binary); err != nil {
		return fmt.Errorf("%q %w", binary, err)
	}

	cmd, env := BuildCmd(c)
	if cmd == nil {
		return fmt.Errorf("buildCommand returned nil")
	}

	if !utils.CanControl(c.Config().GetString("user")) {
		return os.ErrPermission
	}

	// set underlying user for child proc
	username := c.Config().GetString("user")
	errfile := ComponentFilepath(c, "txt")

	if c.Host() != host.LOCAL {
		r := c.Host()
		rUsername := r.GetString("username")
		if rUsername != username && username != "" {
			return fmt.Errorf("cannot run remote process as a different user (%q != %q)", rUsername, username)
		}
		rem, err := r.Dial()
		if err != nil {
			return err
		}
		sess, err := rem.NewSession()
		if err != nil {
			return err
		}

		// we have to convert cmd to a string ourselves as we have to quote any args
		// with spaces (like "Demo Gateway")
		//
		// given this is sent to a shell, we can quote everything blindly ?
		//
		// note that cmd.Args hosts the command as Args[0], so no Path required
		var cmdstr = ""
		for _, a := range cmd.Args {
			cmdstr = fmt.Sprintf("%s %q", cmdstr, a)
		}
		pipe, err := sess.StdinPipe()
		if err != nil {
			return err
		}

		if err = sess.Shell(); err != nil {
			return err
		}
		fmt.Fprintln(pipe, "cd", c.Home())
		for _, e := range env {
			fmt.Fprintln(pipe, "export", e)
		}
		fmt.Fprintf(pipe, "%s > %q 2>&1 &", cmdstr, errfile)
		fmt.Fprintln(pipe, "exit")
		sess.Close()
		// wait a short while for remote to catch-up
		time.Sleep(250 * time.Millisecond)

		pid, err := GetPID(c)
		if err != nil {
			return err
		}
		fmt.Printf("%s started with PID %d\n", c, pid)
		return nil
	}

	// pass possibly empty string down to setuser - it handles defaults
	if err = utils.SetUser(cmd, username); err != nil {
		return
	}

	cmd.Env = append(os.Environ(), env...)

	out, err := os.OpenFile(errfile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	procSetupOS(cmd, out)

	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Dir = c.Home()

	if err = cmd.Start(); err != nil {
		return
	}
	fmt.Printf("%s started with PID %d\n", c, cmd.Process.Pid)
	if cmd.Process != nil {
		// detach from control
		cmd.Process.Release()
	}

	return
}
