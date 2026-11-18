package process

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/go-reap"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

// Program is a highly simplified representation of a program to manage
// with Start or Batch.
//
// Args and Env can be either a comma delimited string, which is split
// by config, or a slice of strings which is used as-is
type Program struct {
	Executable string        `json:"executable,omitempty"` // Path to program, passed through exec.LookPath
	Username   string        `json:"username,omitempty"`   // The name of the user, if empty use current
	WorkingDir string        `json:"workingdir,omitempty"` // The working directory, defaults to home dir of user
	ErrLog     string        `json:"errlog,omitempty"`     // The name of the logfile, defaults to basename of program+".log" in Dir
	Args       []string      `json:"args,omitempty"`       // Args not including program
	Env        []string      `json:"env,omitempty"`        // Env as key=value pairs
	Foreground bool          `json:"foreground,omitempty"` // Run in foreground, to completion, return if result != 0
	Restart    bool          `json:"restart,omitempty"`    // restart on exit
	IgnoreErr  bool          `json:"ignoreerr,omitempty"`  // If true do not return err on failure
	Wait       time.Duration `json:"wait,omitempty"`       // period to wait after starting, default none
}

func retErrIfFalse(ret bool, err error) error {
	if !ret {
		return err
	}
	return nil
}

// newCommand builds the command for program, which is run as another
// user if setCreds is true
func newCommand(program Program, setCreds bool) (*exec.Cmd, error) {
	cmd := exec.Command(program.Executable, program.Args...)
	if setCreds {
		if err := setCredentialsFromUsername(cmd, program.Username); err != nil {
			return nil, err
		}
	}
	cmd.Env = program.Env
	cmd.Dir = program.WorkingDir
	return cmd, nil
}

// homeDir returns the home directory of the user
func homeDir(username string) (dir string, err error) {
	u, err := user.Lookup(username)
	if err == nil {
		return u.HomeDir, nil
	}
	// a user that is not in the user database, which is the case for
	// containers run with an arbitrary UID, can still have a HOME
	if home, herr := os.UserHomeDir(); herr == nil {
		if cu, cerr := user.Current(); cerr != nil || cu.Username == username {
			return home, nil
		}
	}
	return "", err
}

// withDefaultEnv returns env, which is not changed, with each of the
// defaults, which are in the form KEY=VALUE, added at the end only if
// there is no entry for the key already, so that values from the caller
// are used in preference.
func withDefaultEnv(env []string, defaults ...string) []string {
	out := slices.Clone(env)
	for _, d := range defaults {
		key, _, _ := strings.Cut(d, "=")
		if !slices.ContainsFunc(env, func(e string) bool {
			k, _, found := strings.Cut(e, "=")
			return found && k == key
		}) {
			out = append(out, d)
		}
	}
	return out
}

// Start runs a process on host h. It is run detached in the background
// unless Foreground is true. The PID of a process started in the
// background is returned, but not for one run in the foreground, which
// has completed.
//
// Values in Env are used in preference to the defaults for HOME, SHELL,
// USER, LOGNAME and PATH, which are added only if not set.
//
// TODO: return error windows
// TODO: look at remote processes
func Start(h host.Host, program Program, options ...ProgramOption) (pid int, err error) {
	opts := evalOptions(options...)

	if h == nil {
		return 0, fmt.Errorf("host cannot be nil")
	}

	// run as another user, which has to be set using credentials
	var setCreds bool

	if program.Username != "" && program.Username != h.Username() {
		// if username is set and is not current user for host h

		// if not root
		if os.Getuid() != 0 && os.Geteuid() != 0 {
			return 0, os.ErrPermission
		}

		// if root, and not localhost
		if h != host.Localhost {
			return 0, host.ErrInvalidArgs
		}
		u, err := user.Lookup(program.Username)
		if err != nil {
			return 0, err
		}
		if program.WorkingDir == "" {
			program.WorkingDir = u.HomeDir
		}
		setCreds = true
	} else {
		if program.Username == "" {
			if u, err := user.Current(); err == nil {
				program.Username = u.Username
			} else {
				program.Username = os.Getenv("USER")
			}
		}

		if program.WorkingDir == "" {
			if program.WorkingDir, err = homeDir(program.Username); err != nil {
				return 0, err
			}
		}
	}

	// build basic env, keeping PATH, but only for anything the caller
	// has not set
	program.Env = withDefaultEnv(program.Env,
		"HOME="+program.WorkingDir,
		"SHELL=/bin/bash",
		"USER="+program.Username,
		"LOGNAME="+program.Username,
		"PATH="+os.Getenv("PATH"),
	)

	// check for expand options
	if opts.expandArgs {
		program.Args = config.ExpandStringSlice(program.Args, config.LookupTable(opts.lookup...))
	}

	if opts.expandEnv {
		program.Env = config.ExpandStringSlice(program.Env, config.LookupTable(opts.lookup...))
	}

	if program.ErrLog == "" {
		program.ErrLog = path.Join(program.WorkingDir, path.Base(program.Executable+".log"))
	} else if !path.IsAbs(program.ErrLog) {
		program.ErrLog = path.Join(program.WorkingDir, program.ErrLog)
	}

	p, err := exec.LookPath(program.Executable)
	if err != nil {
		return 0, retErrIfFalse(program.IgnoreErr, fmt.Errorf("%q %w", program.Executable, err))
	}

	switch {
	case program.Foreground:
		cmd, err := newCommand(program, setCreds)
		if err != nil {
			return 0, err
		}
		if _, err := h.Run(cmd, host.ProcessErrfile(program.ErrLog)); err != nil {
			return 0, retErrIfFalse(program.IgnoreErr, err)
		}
		return 0, nil

	case program.Restart:
		go func() {
			for {
				cmd, err := newCommand(program, setCreds)
				if err != nil {
					return
				}
				h.Run(cmd, host.ProcessErrfile(program.ErrLog))
				if program.Wait != 0 {
					time.Sleep(program.Wait)
				} else {
					time.Sleep(500 * time.Millisecond)
				}
			}
		}()

		if program.Wait != 0 {
			time.Sleep(program.Wait)
		}

		// only valid if long running, and finds a process, not necessarily this one
		pid, err = PID(h, filepath.Base(p), nil, RefreshCache())
		return pid, retErrIfFalse(program.IgnoreErr, err)

	default:
		cmd, err := newCommand(program, setCreds)
		if err != nil {
			return 0, err
		}
		if pid, err = h.Start(cmd, host.ProcessErrfile(program.ErrLog)); err != nil {
			return 0, retErrIfFalse(program.IgnoreErr, err)
		}
		if program.Wait != 0 {
			time.Sleep(program.Wait)
		}
		return pid, nil
	}
}

// Batch executes the slice of Program entries using Start. If any stage
// returns err then Batch returns immediately. Set IgnoreErr in Program
// to not return errors for each stage. If any stage has Restart set and
// it is supported then a reaper of child processes is run, and the done
// channel is returned, which stops the reaper when closed.
func Batch(h host.Host, batch []Program, options ...ProgramOption) (done chan struct{}, err error) {
	restart := false
	for _, program := range batch {
		if program.Restart {
			restart = true
		}
		_, err = Start(h, program, options...)
		if err != nil && !errors.Is(err, os.ErrProcessDone) {
			return
		}
	}
	if errors.Is(err, os.ErrProcessDone) {
		err = nil
	}
	if restart && reap.IsSupported() {
		done = make(chan struct{})
		go reap.ReapChildren(nil, nil, done, nil)
	}
	return
}

// Daemon backgrounds the current process by re-executing the existing
// binary (as found by [os.Executable], so may there is a small window
// while the referenced binary can change). The function passed as
// processArgs is called with any further arguments passed to it as
// parameters and can be used to remove flags that triggered the
// daemonisation in the first place. A helper function - [RemoveArgs] -
// is available to do this.
//
// The child process is executed with any extraArgs appended to the
// command line.
//
// If successful the function never returns and the child process PID is
// written to writepid, if not nil. Remember to only open the file
// inside the test for daemon mode in the caller, otherwise on
// re-execution the file will be re-opened and overwrite the one from
// the parent. writepid is closed in the parent.
//
// On failure the function does return with an error.
//
//	process.Daemon(os.Stdout, nil, process.RemoveArgs, "-D", "--daemon")
func Daemon(writepid io.WriteCloser, extraArgs []string, processArgs func([]string, ...string) []string, args ...string) (err error) {
	var newargs []string
	bin, err := os.Executable()
	if err != nil {
		return
	}
	if processArgs == nil {
		processArgs = RemoveArgs
	}
	newargs = processArgs(slices.Clone(os.Args[1:]), args...)
	newargs = append(newargs, extraArgs...)
	cmd := exec.Command(bin, newargs...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// OS specific (compile time/build constraint) changes to cmd
	prepareCmd(cmd)

	if err = cmd.Start(); err != nil {
		return
	}

	// write the resulting PID to writepid if non-nil. close writepid
	if writepid != nil {
		fmt.Fprintln(writepid, cmd.Process.Pid)
		writepid.Close()
	}
	if cmd.Process != nil {
		cmd.Process.Release()
	}
	os.Exit(0)
	return // not reached
}

// RemoveArgs is a helper function for Daemon(). Daemon calls the
// function with a copy of os.Args[1:] and removes any arguments matching
// members of the slice `remove` and returns the result. Only bare
// arguments are removed and no pattern matching or adjacent values are
// removed. If more complex tests are required then pass Daemon() your
// own function. The slice passed in is not changed.
func RemoveArgs(in []string, remove ...string) (out []string) {
	return slices.DeleteFunc(slices.Clone(in), func(a string) bool {
		return slices.Contains(remove, a)
	})
}
