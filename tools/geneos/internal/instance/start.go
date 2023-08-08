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
	"os/exec"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Start runs the instance.
func Start(i geneos.Instance) (err error) {
	if IsRunning(i) {
		return geneos.ErrRunning
	}

	if IsDisabled(i) {
		return geneos.ErrDisabled
	}

	// changing users is not supported
	username := i.Host().Username()
	instanceUsername := i.Config().GetString("user")

	if instanceUsername != "" && username != instanceUsername {
		return fmt.Errorf("%s is configured with a different user to the one trying to start it (instance user %q != %q (you))", i, instanceUsername, username)
	}

	binary := i.Config().GetString("program")
	if _, err = i.Host().Stat(binary); err != nil {
		return fmt.Errorf("%q %w", binary, err)
	}

	cmd, env, home := BuildCmd(i, false)
	if cmd == nil {
		return fmt.Errorf("BuildCmd() returned nil")
	}

	// set underlying user for child proc
	errfile := ComponentFilepath(i, "txt")

	i.Host().Start(cmd, env, home, errfile)
	pid, err := GetPID(i)
	if err != nil {
		return err
	}
	fmt.Printf("%s started with PID %d\n", i, pid)
	return nil
}

// BuildCmd gathers the path to the binary, arguments and any
// environment variables for an instance and returns an exec.Cmd, almost
// ready for execution. Callers will add more details such as working
// directories, user and group etc.
//
// If noDecode is set then any secure environment variables are not decoded,
// so can be used for display
func BuildCmd(i geneos.Instance, noDecode bool) (cmd *exec.Cmd, env []string, home string) {
	binary := PathOf(i, "program")

	args, env, home := i.Command()

	opts := strings.Fields(i.Config().GetString("options"))
	args = append(args, opts...)

	envs := i.Config().GetStringSlice("env", config.NoDecode(noDecode))
	libs := []string{}
	if i.Config().GetString("libpaths") != "" {
		libs = append(libs, i.Config().GetString("libpaths"))
	}

	for _, e := range envs {
		switch {
		case strings.HasPrefix(e, "LD_LIBRARY_PATH="):
			libs = append(libs, strings.TrimPrefix(e, "LD_LIBRARY_PATH="))
		default:
			env = append(env, e)
		}
	}
	if len(libs) > 0 {
		env = append(env, "LD_LIBRARY_PATH="+strings.Join(libs, ":"))
	}
	cmd = exec.Command(binary, args...)

	return
}
