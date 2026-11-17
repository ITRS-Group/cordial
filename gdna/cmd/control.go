/*
Copyright © 2024 ITRS Group

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

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/pkg/process/control"
)

const (
	// pidFileKey is the configuration key that sets the PID file
	pidFileKey = "gdna.pidfile"

	// pidFileName is the name of the PID file, in the default directory,
	// if no other is set
	pidFileName = "gdna.pid"
)

// pidFile is the value of the `--pidfile` flag
var pidFile string

// background is the control of `gdna start` when it is running in the
// background, with the commands `stop`, `restart`, `reload` and `status`
var background = &control.Daemon{
	Name:        "gdna",
	PIDFileName: pidFileName,
	PIDFile:     &pidFile,
	PIDFileKey:  pidFileKey,
	ConfigFile:  func() string { return cfgFile },
	Load:        loadConfig,
	Check:       checkSchedules,
	StopHelp: `The process is asked to stop and it then cancels any fetch or report that is
running before it exits.`,
	ReloadHelp: `A fetch or report that is running when the configuration is reloaded
completes using the configuration it started with, and the schedules and all
later runs use the new one. Changes to the database, the logging and the PID
file need a restart.`,
}

func init() {
	for _, c := range background.Commands() {
		// these commands only use the PID file, and must not rotate or
		// write to the log of the process they control
		c.Annotations = map[string]string{"defaultlog": os.DevNull}
		c.DisableAutoGenTag = true
		c.DisableFlagsInUseLine = true
		Cmd.AddCommand(c)
	}
}

// pidFilePath returns the path of the PID file for a process started
// with the configuration cf, which is the one given by the `--pidfile`
// flag or the configuration or, if neither is set, the default
func pidFilePath(cf *config.Config) string {
	if path := control.ConfiguredPIDFileKey(cf, pidFile, pidFileKey); path != "" {
		return path
	}
	return control.DefaultPIDFile(pidFileName)
}

// runningConfigFile returns the absolute path of the configuration file
// in use, which is recorded in the PID file so that the control commands
// check, and the process reloads, the same file wherever they are run
// from. It is empty if only the internal defaults are in use.
func runningConfigFile() string {
	file := cfgFile

	// a file given on the command line is always used, and if it has gone
	// since start-up then loading it, for a reload, must fail and not
	// silently use the defaults
	if file == "" {
		file = config.Path(execname, append(configOptions(cfgFile), config.MustExist())...)

		// there is no file if the result is a description, such as
		// "internal defaults", and not a path that exists
		if _, err := os.Stat(file); file == "" || err != nil {
			return ""
		}
	}
	if abs, err := filepath.Abs(file); err == nil {
		return abs
	}
	return file
}

// writePIDFile records how the process was started in the PID file,
// which says that it is ready
func writePIDFile(pf *process.PIDFile) error {
	info := process.PIDInfo{
		Args:       os.Args[1:],
		ConfigFile: runningConfigFile(),
	}
	info.Executable, _ = os.Executable()
	info.Dir, _ = os.Getwd()
	if err := pf.Write(info); err != nil {
		return fmt.Errorf("cannot write pid file: %w", err)
	}
	return nil
}
