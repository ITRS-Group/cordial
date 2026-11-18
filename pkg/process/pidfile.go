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

package process

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// PIDInfo is the content of a PID file. The PID file records how the
// process was started so that it can be restarted in the same way.
type PIDInfo struct {
	PID        int       `json:"pid"`
	Executable string    `json:"executable,omitempty"`
	Args       []string  `json:"args,omitempty"` // arguments, not including the executable
	Dir        string    `json:"dir,omitempty"`  // working directory
	ConfigFile string    `json:"config_file,omitempty"`
	Started    time.Time `json:"started,omitzero"`
}

// PIDFileStatus is the state of a PID file
type PIDFileStatus struct {
	// Info is the content of the PID file, which can be the zero value
	// if the file is empty (the process is still starting up) or does
	// not exist
	Info PIDInfo
	// Running is true if the process that created the file is running.
	// A PID file left by a process that did not exit cleanly is not
	// running.
	Running bool
	// Exists is true if the file exists
	Exists bool
}

// AlreadyRunningError is returned when a PID file is held by a running
// process
type AlreadyRunningError struct {
	Path string
	PID  int
}

func (e *AlreadyRunningError) Error() string {
	if e.PID != 0 {
		return fmt.Sprintf("already running (pid %d, pid file %s)", e.PID, e.Path)
	}
	return fmt.Sprintf("already running (pid file %s)", e.Path)
}

// PIDFile is a PID file held by the running process, which holds a lock
// on the file for as long as it is running. This means that the file
// cannot be mistaken for a live process after a crash or a reused PID,
// and that a second instance cannot be started using the same file.
type PIDFile struct {
	path string
	f    *os.File
}

// LockPIDFile creates, if required, and locks the PID file at path,
// returning [AlreadyRunningError] if another process holds it. The file
// is empty until [PIDFile.Write] is called, which should only be after
// the process is ready to do its work.
func LockPIDFile(path string) (*PIDFile, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return lockPIDFile(path)
}

// Path returns the path of the file
func (p *PIDFile) Path() string { return p.path }

// Write writes info to the file. If info.PID is zero then the PID of
// the current process is used and if info.Started is zero then the
// current time is used.
func (p *PIDFile) Write(info PIDInfo) error {
	if info.PID == 0 {
		info.PID = os.Getpid()
	}
	if info.Started.IsZero() {
		info.Started = time.Now().Truncate(time.Second)
	}
	b, err := json.Marshal(info)
	if err != nil {
		return err
	}
	if err = p.f.Truncate(0); err != nil {
		return err
	}
	if _, err = p.f.WriteAt(append(b, '\n'), 0); err != nil {
		return err
	}
	return p.f.Sync()
}

// Release removes the file and releases the lock
func (p *PIDFile) Release() error {
	// remove first, while we still hold the lock, so that nothing can
	// take it and have its file removed
	err := os.Remove(p.path)
	if cerr := p.f.Close(); err == nil {
		err = cerr
	}
	return err
}

// ReadPIDFile returns the content of the PID file. A file that is empty,
// which is the case while the process is starting, returns the zero
// value and no error. A file that contains just a PID, as written by
// other programs, is accepted.
func ReadPIDFile(path string) (info PIDInfo, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if len(b) == 0 {
		return
	}
	if json.Unmarshal(b, &info) != nil {
		if _, serr := fmt.Sscanf(string(b), "%d", &info.PID); serr != nil {
			return PIDInfo{}, fmt.Errorf("pid file %s is not valid", path)
		}
	}
	return
}

// StatusOfPIDFile reports the state of the process that holds the PID
// file at path. It is not an error for the file to not exist.
func StatusOfPIDFile(path string) (status PIDFileStatus, err error) {
	info, err := ReadPIDFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return status, nil
	}
	if err != nil {
		return
	}
	status.Exists = true
	status.Info = info
	status.Running, err = pidFileHeld(path, info)
	return
}

// WaitStopped waits until the process that holds the PID file at path
// has exited, for up to timeout, and returns true if it has.
func WaitStopped(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if st, err := StatusOfPIDFile(path); err == nil && !st.Running {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// StartDetached starts the program in its own session (or process group
// on Windows), detached from the caller, with no input or output, and
// returns it without waiting for it. The caller can use [WaitReady] to
// find out if it started.
func StartDetached(executable string, args []string, dir string) (cmd *exec.Cmd, err error) {
	cmd = exec.Command(executable, args...)
	cmd.Dir = dir
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// OS specific (compile time/build constraint) changes to cmd
	prepareCmd(cmd)

	if err = cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// WaitReady waits, for up to timeout, until cmd, which was started with
// [StartDetached], has written its PID to the PID file at path, which
// is how a process says it is ready. It returns an error if cmd exits
// first or the timeout expires.
func WaitReady(cmd *exec.Cmd, path string, timeout time.Duration) error {
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.After(timeout)

	for {
		select {
		case err := <-exited:
			if err != nil {
				return fmt.Errorf("process exited before it was ready: %w", err)
			}
			return errors.New("process exited before it was ready")
		case <-ticker.C:
			if st, err := StatusOfPIDFile(path); err == nil && st.Running && st.Info.PID == cmd.Process.Pid {
				return nil
			}
		case <-deadline:
			return fmt.Errorf("process (pid %d) was not ready after %s", cmd.Process.Pid, timeout)
		}
	}
}

// SignalProcess sends the signal to the process with the PID, which must
// be greater than zero as anything else addresses a group of processes.
// It returns [os.ErrProcessDone] if there is no such process.
func SignalProcess(pid int, sig os.Signal) error {
	if sig == nil {
		return errors.ErrUnsupported
	}
	if pid <= 0 {
		return fmt.Errorf("invalid pid %d: %w", pid, os.ErrInvalid)
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return os.ErrProcessDone
	}
	return p.Signal(sig)
}
