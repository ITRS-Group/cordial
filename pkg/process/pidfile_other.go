//go:build !(linux || darwin || freebsd || netbsd || openbsd || dragonfly)

package process

import (
	"os"
)

// lockPIDFile creates the file, which is not locked on this platform. A
// file for a process that is still running is an error and one for a
// process that is not is replaced.
func lockPIDFile(path string) (*PIDFile, error) {
	if info, err := ReadPIDFile(path); err == nil && info.PID != 0 && info.PID != os.Getpid() && processAlive(info.PID) {
		return nil, &AlreadyRunningError{Path: path, PID: info.PID}
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	return &PIDFile{path: path, f: f}, nil
}

// pidFileHeld is true if the process in the file is running. As there is
// no lock a reused PID cannot be detected.
func pidFileHeld(path string, info PIDInfo) (bool, error) {
	if info.PID == 0 {
		// the file is empty, so the process is starting, but we cannot
		// tell if it still is
		return false, nil
	}
	return processAlive(info.PID), nil
}
