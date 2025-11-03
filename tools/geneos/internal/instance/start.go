/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package instance

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
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
	cmd, err := BuildCmd(i, false, options...)
	if err != nil {
		return
	}
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
	pid, err := GetLivePID(i)
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
// so can be used for display to a user without revealing secrets.
//
// Any extras arguments are appended without further checks
func BuildCmd(i geneos.Instance, noDecode bool, options ...StartOptions) (cmd *exec.Cmd, err error) {
	var env []string
	var home string

	so := evalStartOptions(options...)

	binary := PathOf(i, "program")

	args, env, home, err := i.Command(so.skipfilecheck)
	if err != nil {
		return
	}

	opts := strings.Fields(i.Config().GetString("options"))
	args = append(args, opts...)

	args = append(args, so.extras...)

	libs := []string{}
	if i.Config().GetString("libpaths") != "" {
		libs = append(libs, i.Config().GetString("libpaths"))
	}

	for _, e := range i.Config().GetStringSlice("env", config.NoDecode(noDecode)) {
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

	// pass through the user's home directory unless there is one specifically defined
	if !slices.ContainsFunc(env, func(e string) bool {
		return strings.HasPrefix(e, "HOME=")
	}) {
		env = append(env, "HOME="+os.Getenv("HOME"))
	}

	// similarly, check for a PATH setting, else use a very plain one.
	// this can be overridden by the user using the `-e PATH=...` option
	// to start and restart commands
	if !slices.ContainsFunc(env, func(e string) bool {
		return strings.HasPrefix(e, "PATH=")
	}) {
		if strings.Contains(i.Host().ServerVersion(), "windows") {
			p := "C:\\Windows\\System32;C:\\Windows;C:\\Windows\\System32\\Wbem"
			up, ok := os.LookupEnv("USERPROFILE")
			if ok {
				p = fmt.Sprintf("%s%s;%s", p, up+"\\bin", up+"\\AppData\\Local\\Microsoft\\WindowsApps")
			}
			env = append(env, p)
		} else {
			env = append(env, "PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin")
		}
	}

	cmd = exec.Command(binary, args...)
	cmd.Env = env
	cmd.Dir = home

	return
}

type startOptions struct {
	envs          []string
	extras        []string
	skipfilecheck bool
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

func SkipFileCheck() StartOptions {
	return func(so *startOptions) {
		so.skipfilecheck = true
	}
}
