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
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/rs/zerolog/log"
)

// Start runs the instance.
func Start(i geneos.Instance, opts ...any) (err error) {
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

	options := []StartOptions{}
	for _, o := range opts {
		if option, ok := o.(StartOptions); ok {
			options = append(options, option)
		}
	}
	cmd := BuildCmd(i, false, options...)
	if cmd == nil {
		return fmt.Errorf("BuildCmd() returned nil")
	}

	// set underlying user for child proc
	errfile := ComponentFilepath(i, "txt")

	log.Debug().Msgf("starting '%s'", cmd.String())
	if err = i.Host().Start(cmd, errfile); err != nil {
		return
	}
	// wait a bit for the process to start before checking
	time.Sleep(250 * time.Millisecond)
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
//
// Any extras arguments are appended without further checks
func BuildCmd(i geneos.Instance, noDecode bool, options ...StartOptions) (cmd *exec.Cmd) {
	var env []string
	var home string

	binary := PathOf(i, "program")

	args, env, home := i.Command()

	opts := strings.Fields(i.Config().GetString("options"))
	args = append(args, opts...)

	so := evalStartOptions(options...)
	args = append(args, so.extras...)

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
	env = append(env, so.envs...)

	cmd = exec.Command(binary, args...)
	cmd.Env = env
	cmd.Dir = home

	return
}

type startOptions struct {
	envs   []string
	extras []string
}

type StartOptions func(*startOptions)

func evalStartOptions(options ...StartOptions) (d *startOptions) {
	// defaults
	d = &startOptions{}
	for _, opt := range options {
		opt(d)
	}
	return
}

// StartingExtras sets extra command line parameters by splitting extras
// on spaces. Quotes, escaping and other shell-like separators are
// ignored.
func StartingExtras(extras string) StartOptions {
	return func(so *startOptions) {
		so.extras = strings.Fields(extras)
	}
}

// StartingEnvs takes a NameValues list of extra environment variables
// to append to the standard list for the instance.
func StartingEnvs(envs NameValues) StartOptions {
	return func(so *startOptions) {
		so.envs = envs
	}
}
