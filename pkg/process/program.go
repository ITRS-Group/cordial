package process

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path"
	"slices"
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

// Start runs a process on host h. It is run detached in the background
// unless Foreground is true.
//
// TODO: return error windows
// TODO: look at remote processes
func Start(h host.Host, program Program, options ...ProgramOption) (pid int, err error) {
	opts := evalOptions(options...)

	if h == nil {
		return 0, fmt.Errorf("host cannot be nil")
	}

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
	} else {
		if program.Username == "" {
			if u, err := user.Current(); err == nil {
				program.Username = u.Username
			} else {
				program.Username = os.Getenv("USER")
			}
		}

		if program.WorkingDir == "" {
			u, _ := user.Lookup(program.Username)
			program.WorkingDir = u.HomeDir
		}
	}

	// build basic env after any caller values set, keep PATH
	program.Env = append(program.Env,
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
		return 0, retErrIfFalse(program.IgnoreErr, fmt.Errorf("%q %w", p, err))
	}

	switch {
	case program.Foreground:
		cmd := exec.Command(program.Executable, program.Args...)
		setCredentialsFromUsername(cmd, program.Username)
		cmd.Env = program.Env
		cmd.Dir = program.WorkingDir
		if _, err := h.Run(cmd, host.ProcessErrfile(program.ErrLog)); err != nil {
			return 0, retErrIfFalse(program.IgnoreErr, err)
		}
	case program.Restart:
		go reap.ReapChildren(nil, nil, nil, nil)
		go func() {
			for {
				cmd := exec.Command(program.Executable, program.Args...)
				setCredentialsFromUsername(cmd, program.Username)
				cmd.Env = program.Env
				cmd.Dir = program.WorkingDir
				h.Run(cmd, host.ProcessErrfile(program.ErrLog))
				if program.Wait != 0 {
					time.Sleep(program.Wait)
				} else {
					time.Sleep(500 * time.Millisecond)
				}
			}
		}()
	default:
		cmd := exec.Command(program.Executable, program.Args...)
		setCredentialsFromUsername(cmd, program.Username)
		cmd.Env = program.Env
		cmd.Dir = program.WorkingDir
		if err = h.Start(cmd, host.ProcessErrfile(program.ErrLog)); err != nil {
			return 0, retErrIfFalse(program.IgnoreErr, err)
		}
	}

	if program.Wait != 0 {
		time.Sleep(program.Wait)
	}

	// only valid if long running
	pid, err = PID(h, p, []string{}, RefreshCache())
	err = retErrIfFalse(program.IgnoreErr, err)
	return
}

// Batch executes the slice of Program entries using Start. If any stage
// returns err then Batch returns immediately. Set IgnoreErr in Program
// to not return errors for each stage. If any stage has Restart set and
// it is supported then a reaper is run and the done channel returned.
func Batch(h host.Host, batch []Program, options ...ProgramOption) (done chan struct{}, err error) {
	r := false
	for _, program := range batch {
		if program.Restart {
			r = true
		}
		_, err = Start(h, program, options...)
		if err != nil && err != os.ErrProcessDone {
			return
		}
	}
	if r && reap.IsSupported() {
		go reap.ReapChildren(nil, nil, done, nil)
	}

	if err == os.ErrProcessDone {
		err = nil
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
// If successful the function never returns and the child process PID is
// written to writepid, if not nil. Remember to only open the file
// inside the test for daemon mode in the caller, otherwise on
// re-execution the file will be re-opened and overwrite the one from
// the parent. writepid is closed in the parent.
//
// On failure the function does return with an error.
//
//	process.Daemon(os.Stdout, process.RemoveArgs, "-D", "--daemon")
func Daemon(writepid io.WriteCloser, processArgs func([]string, ...string) []string, args ...string) (err error) {
	bin, err := os.Executable()
	if err != nil {
		return
	}
	var newargs []string
	if processArgs == nil {
		newargs = RemoveArgs(os.Args[1:], args...)
	} else {
		newargs = processArgs(os.Args[1:], args...)
	}
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

// Daemon2 backgrounds the current process by re-executing the existing
// binary (as found by [os.Executable], so may there is a small window
// while the referenced binary can change). The function passed as
// processArgs is called with any further arguments passed to it as
// parameters and can be used to remove flags that triggered the
// daemonisation in the first place. A helper function - [RemoveArgs] -
// is available to do this.
//
// The child process is executed with any `addArgs` appended to the
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
//	process.Daemon(os.Stdout, process.RemoveArgs, "-D", "--daemon")
func Daemon2(writepid io.WriteCloser, addArgs []string, processArgs func([]string, ...string) []string, args ...string) (err error) {
	bin, err := os.Executable()
	if err != nil {
		return
	}
	var newargs []string
	if processArgs == nil {
		newargs = RemoveArgs(os.Args[1:], args...)
	} else {
		newargs = processArgs(os.Args[1:], args...)
	}
	newargs = append(newargs, addArgs...)
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
// function with os.Args[1:] and removes any arguments matching members
// of the slice `remove` and returns the result. Only bare arguments are
// removed and no pattern matching or adjacent values are removed. If
// more complex tests are required then pass Daemon() your own function.
func RemoveArgs(in []string, remove ...string) (out []string) {
	out = slices.DeleteFunc(in, func(a string) bool {
		if slices.Contains(remove, a) {
			return true
		}
		return false
	})

	return
}
