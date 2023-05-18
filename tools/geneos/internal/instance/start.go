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

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Start runs the instance.
func Start(c geneos.Instance) (err error) {
	if IsRunning(c) {
		return geneos.ErrRunning
	}

	if IsDisabled(c) {
		return geneos.ErrDisabled
	}

	// changing users is not supported
	username := c.Host().Username()
	instanceUsername := c.Config().GetString("user")

	if instanceUsername != "" && username != instanceUsername {
		return fmt.Errorf("%s is configured with a different user to the one trying to start it (instance user %q != %q (you))", c, instanceUsername, username)
	}

	binary := c.Config().GetString("program")
	if _, err = c.Host().Stat(binary); err != nil {
		return fmt.Errorf("%q %w", binary, err)
	}

	cmd, env := BuildCmd(c)
	if cmd == nil {
		return fmt.Errorf("BuildCmd() returned nil")
	}

	// set underlying user for child proc
	errfile := ComponentFilepath(c, "txt")

	c.Host().Start(cmd, env, c.Home(), errfile)
	pid, err := GetPID(c)
	if err != nil {
		return err
	}
	fmt.Printf("%s started with PID %d\n", c, pid)
	return nil
}
