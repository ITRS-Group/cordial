/*
Copyright © 2026 ITRS Group

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

// Package control provides the `stop`, `restart`, `reload` and `status`
// commands for programs that run a server in the background using
// [ims.ServeWith], and the support for starting them in the background.
//
// The commands find the server using the PID file that it holds while
// running. The PID file also records how the server was started, so that
// it can be restarted in the same way, and which configuration file it
// uses, so that the configuration can be checked before a reload.
package control

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/itrs-group/cordial/pkg/process"
)

const (
	// DefaultStopTimeout is how long to wait for a server to stop. It is
	// longer than the time the server gives requests in progress.
	DefaultStopTimeout = 30 * time.Second

	// readyTimeout is how long to wait for a server that has been
	// started to be ready
	readyTimeout = 15 * time.Second
)

// Daemon describes a program that runs a server in the background
type Daemon struct {
	// Name is the description of the program in messages, for example
	// `servicenow2 proxy`
	Name string

	// PIDFileName is the file name of the PID file in the default
	// directory, used if no other PID file is configured
	PIDFileName string

	// PIDFile points to the value of the program's `--pidfile` flag,
	// which the commands also add
	PIDFile *string

	// ConfigFile returns the configuration file given on the command
	// line, which is empty if it was not
	ConfigFile func() string

	// Load reads the server configuration from the file, which is
	// empty to use the default location
	Load func(file string) (*config.Config, error)

	// Check does any checks of a configuration that are specific to the
	// program, in addition to those of the server, and is optional
	Check func(cf *config.Config) error
}

// FlagUsage is the help text for the `--pidfile` flag, for the program to
// use on its own start command
const FlagUsage = "Use the PID `file`, overriding `server.pidfile` in the configuration"

// DefaultPIDFile returns the path of the PID file in the default
// directory, which is `geneos` in the user cache directory
func DefaultPIDFile(name string) string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "geneos", name)
}

// ConfiguredPIDFile returns the path of the PID file from the flag or,
// if not given, the `server.pidfile` setting in the configuration. It
// is empty if neither is set.
func ConfiguredPIDFile(cf *config.Config, flag string) string {
	if flag != "" {
		return flag
	}
	if cf != nil {
		return config.Get[string](cf, cf.Join("server", "pidfile"))
	}
	return ""
}

// Daemonize starts the program again, in the background, with the same
// arguments less the `-D` and `--daemon` flags, plus extraArgs and the
// PID file, and waits for it to be ready, which is when it has started
// listening and holds the PID file. It returns the PID of the new
// process, or an error if it could not be started or did not become
// ready, in which case the log should be consulted.
func Daemonize(pidFile string, extraArgs ...string) (pid int, err error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}
	dir, _ := os.Getwd()

	args := process.RemoveArgs(slices.Clone(os.Args[1:]), "-D", "--daemon")
	args = append(args, extraArgs...)
	args = append(args, "--pidfile", pidFile)

	cmd, err := process.StartDetached(exe, args, dir)
	if err != nil {
		return 0, err
	}
	if err = process.WaitReady(cmd, pidFile, readyTimeout); err != nil {
		return 0, err
	}
	return cmd.Process.Pid, nil
}

// Commands returns the `stop`, `restart`, `reload` and `status`
// commands.
func (d *Daemon) Commands() []*cobra.Command {
	return []*cobra.Command{d.stopCmd(), d.restartCmd(), d.reloadCmd(), d.statusCmd()}
}

// pidFile returns the path of the PID file for the commands
func (d *Daemon) pidFile() string {
	if d.PIDFile != nil && *d.PIDFile != "" {
		return *d.PIDFile
	}

	// try the configuration, which can fail without it being a problem
	// as the default is then used, but then the default configuration
	// must be the one being used
	if d.Load != nil {
		file := ""
		if d.ConfigFile != nil {
			file = d.ConfigFile()
		}
		if cf, err := d.Load(file); err == nil {
			if p := ConfiguredPIDFile(cf, ""); p != "" {
				return p
			}
		}
	}
	return DefaultPIDFile(d.PIDFileName)
}

func (d *Daemon) addFlags(c *cobra.Command) {
	if d.PIDFile != nil {
		c.Flags().StringVar(d.PIDFile, "pidfile", "", FlagUsage)
	}
}

type stopOptions struct {
	timeout time.Duration
	force   bool
}

func (o *stopOptions) addFlags(c *cobra.Command) {
	c.Flags().DurationVar(&o.timeout, "timeout", DefaultStopTimeout, "Wait this long for the process to stop")
	c.Flags().BoolVar(&o.force, "force", false, "Kill the process if it has not stopped within the timeout")
}

// stop stops the process that holds the PID file, and returns the status
// before it was stopped.
func (d *Daemon) stop(out io.Writer, path string, opts stopOptions) (before process.PIDFileStatus, err error) {
	before, err = waitStarted(path)
	if err != nil {
		return
	}
	if !before.Running {
		if before.Exists {
			// left by a process that did not exit cleanly
			os.Remove(path)
		}
		fmt.Fprintf(out, "%s is not running (pid file %s)\n", d.Name, path)
		return
	}
	pid := before.Info.PID

	fmt.Fprintf(out, "stopping %s (pid %d)\n", d.Name, pid)
	err = process.SignalProcess(pid, process.SigTerm)
	switch {
	case errors.Is(err, os.ErrProcessDone):
		err = nil
	case errors.Is(err, syscall.EPERM):
		return before, fmt.Errorf("cannot stop pid %d: %w", pid, os.ErrPermission)
	case err != nil:
		return before, err
	}

	if !process.WaitStopped(path, opts.timeout) {
		if !opts.force {
			return before, fmt.Errorf("pid %d did not stop within %s, use --force to kill it", pid, opts.timeout)
		}
		fmt.Fprintf(out, "killing pid %d\n", pid)
		if err = process.SignalProcess(pid, process.SigKill); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return before, err
		}
		if !process.WaitStopped(path, 5*time.Second) {
			return before, fmt.Errorf("pid %d could not be killed", pid)
		}
	}

	// a process that was killed does not remove its file
	os.Remove(path)
	fmt.Fprintf(out, "%s stopped\n", d.Name)
	return before, nil
}

// waitStarted returns the status of the PID file, waiting if it is held
// by a process that has not yet written its PID, which is while it is
// starting up
func waitStarted(path string) (st process.PIDFileStatus, err error) {
	deadline := time.Now().Add(readyTimeout)
	for {
		st, err = process.StatusOfPIDFile(path)
		if err != nil || !st.Running || st.Info.PID != 0 || time.Now().After(deadline) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (d *Daemon) stopCmd() *cobra.Command {
	var opts stopOptions
	c := &cobra.Command{
		Use:   "stop",
		Short: "Stop the background " + d.Name,
		Long: `Stop the ` + d.Name + ` that is running in the background.

The process is asked to stop, and it then stops accepting requests and
waits for any that are in progress to complete before it exits. If it has
not stopped within the timeout then this command fails, unless --force is
used, which kills it.

The process is found using its PID file.`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := d.stop(cmd.OutOrStdout(), d.pidFile(), opts)
			return err
		},
	}
	d.addFlags(c)
	opts.addFlags(c)
	return c
}

func (d *Daemon) restartCmd() *cobra.Command {
	var opts stopOptions
	c := &cobra.Command{
		Use:   "restart",
		Short: "Restart the background " + d.Name,
		Long: `Stop the ` + d.Name + ` that is running in the background, as the stop command
does, and start it again in the background, with the same arguments as it
was started with, which are recorded in its PID file. This also uses a new
version of the program if it has been upgraded.

To pick up a changed configuration without stopping, use the reload
command instead.`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			path := d.pidFile()

			// the PID file keeps the details, even if the process died
			st, err := waitStarted(path)
			if err != nil {
				return err
			}
			if st.Info.Executable == "" && len(st.Info.Args) == 0 {
				return fmt.Errorf("cannot restart as there is no record of how %s was started, start it with the --daemon flag", d.Name)
			}

			if _, err = d.stop(out, path, opts); err != nil {
				return err
			}

			// use the program that is running this command, which is the
			// new version if there has been an upgrade
			exe, err := os.Executable()
			if err != nil {
				exe = st.Info.Executable
			}
			started, err := process.StartDetached(exe, st.Info.Args, st.Info.Dir)
			if err != nil {
				return err
			}
			if err = process.WaitReady(started, path, readyTimeout); err != nil {
				return fmt.Errorf("%s did not start, see its log: %w", d.Name, err)
			}
			fmt.Fprintf(out, "%s started (pid %d)\n", d.Name, started.Process.Pid)
			return nil
		},
	}
	d.addFlags(c)
	opts.addFlags(c)
	return c
}

func (d *Daemon) reloadCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "reload",
		Short: "Reload the configuration of the background " + d.Name,
		Long: `Make the ` + d.Name + ` that is running in the background read its configuration
again, without stopping it.

The configuration is first checked by this command, using the file given by
the --conf flag or, if there is not one, the file that the process was
started with, and the process is only told to reload if it is usable.
The process checks it again and, if it is not usable, continues with the
configuration it has. Requests that are in progress complete using the
configuration they started with and all others use the new one. The log
of the process shows the result.

Reload is not available on Windows, use restart.`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			path := d.pidFile()

			st, err := waitStarted(path)
			if err != nil {
				return err
			}
			if !st.Running {
				return fmt.Errorf("%s is not running (pid file %s)", d.Name, path)
			}

			file := ""
			if d.ConfigFile != nil {
				file = d.ConfigFile()
			}
			if file == "" {
				file = st.Info.ConfigFile
			}
			cf, err := d.Load(file)
			if err != nil {
				return fmt.Errorf("configuration cannot be loaded, not reloading: %w", err)
			}
			if err = ims.CheckServer(cf); err != nil {
				return fmt.Errorf("configuration is not usable, not reloading: %w", err)
			}
			if d.Check != nil {
				if err = d.Check(cf); err != nil {
					return fmt.Errorf("configuration is not usable, not reloading: %w", err)
				}
			}

			if err = process.SignalProcess(st.Info.PID, process.SigReload); err != nil {
				if errors.Is(err, errors.ErrUnsupported) {
					return errors.New("reload is not supported on this platform, use restart")
				}
				return err
			}
			fmt.Fprintf(out, "reload requested of %s (pid %d), see its log for the result\n", d.Name, st.Info.PID)
			return nil
		},
	}
	d.addFlags(c)
	return c
}

func (d *Daemon) statusCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "status",
		Short: "Show whether the background " + d.Name + " is running",
		Long: `Show whether the ` + d.Name + ` is running in the background, using its PID file.
The exit code is zero only if it is running.`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			path := d.pidFile()
			st, err := waitStarted(path)
			if err != nil {
				return err
			}
			if !st.Running {
				fmt.Fprintf(out, "%s is not running (pid file %s)\n", d.Name, path)
				return errNotRunning
			}
			fmt.Fprintf(out, "%s is running (pid %d, started %s)\n", d.Name, st.Info.PID, st.Info.Started.Format(time.RFC3339))
			if st.Info.ConfigFile != "" {
				fmt.Fprintf(out, "configuration: %s\n", st.Info.ConfigFile)
			}
			return nil
		},
	}
	d.addFlags(c)
	return c
}

var errNotRunning = errors.New("not running")
