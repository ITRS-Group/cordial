package instance

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
)

func Start(c geneos.Instance) (err error) {
	pid, err := GetPID(c)
	if err == nil {
		log.Println(c, "already running with PID", pid)
		return
	}

	if IsDisabled(c) {
		return geneos.ErrDisabled
	}

	binary := c.V().GetString("program")
	if _, err = c.Host().Stat(binary); err != nil {
		return fmt.Errorf("%q %w", binary, err)
	}

	cmd, env := BuildCmd(c)
	if cmd == nil {
		return fmt.Errorf("buildCommand returned nil")
	}

	if !utils.CanControl(c.V().GetString("user")) {
		return os.ErrPermission
	}

	// set underlying user for child proc
	username := c.V().GetString("user")
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
		log.Println(c, "started with PID", pid)
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

	// if we've set-up privs at all, set the redirection output file to the same
	if cmd.SysProcAttr != nil && cmd.SysProcAttr.Credential != nil {
		if err = out.Chown(int(cmd.SysProcAttr.Credential.Uid), int(cmd.SysProcAttr.Credential.Gid)); err != nil {
			log.Println("chown:", err)
		}
	}
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Dir = c.Home()
	// detach process by creating a session (fixed start + log)
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true

	if err = cmd.Start(); err != nil {
		return
	}
	log.Println(c, "started with PID", cmd.Process.Pid)
	if cmd.Process != nil {
		// detach from control
		cmd.Process.Release()
	}

	return
}
