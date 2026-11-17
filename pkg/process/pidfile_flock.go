//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package process

import (
	"errors"
	"os"
	"syscall"
)

func tryFlock(f *os.File, how int) (bool, error) {
	for {
		err := syscall.Flock(int(f.Fd()), how|syscall.LOCK_NB)
		switch {
		case err == nil:
			return true, nil
		case errors.Is(err, syscall.EINTR):
			continue
		case errors.Is(err, syscall.EWOULDBLOCK):
			return false, nil
		default:
			return false, err
		}
	}
}

// lockPIDFile takes an exclusive lock on the file, which is held until
// the process exits or the file is closed
func lockPIDFile(path string) (*PIDFile, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	ok, err := tryFlock(f, syscall.LOCK_EX)
	if err != nil {
		f.Close()
		return nil, err
	}
	if !ok {
		info, _ := ReadPIDFile(path)
		f.Close()
		return nil, &AlreadyRunningError{Path: path, PID: info.PID}
	}
	if err = f.Truncate(0); err != nil {
		f.Close()
		return nil, err
	}
	return &PIDFile{path: path, f: f}, nil
}

// pidFileHeld is true if another process holds the lock on the file
func pidFileHeld(path string, _ PIDInfo) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()

	// a shared lock can only be taken if there is no exclusive lock
	ok, err := tryFlock(f, syscall.LOCK_SH)
	if err != nil {
		return false, err
	}
	return !ok, nil
}
