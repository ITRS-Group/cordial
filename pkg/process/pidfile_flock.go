//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package process

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
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

// afterOpenHook is called by lockPIDFile between opening the file and
// locking it, which is when it can be replaced, so that tests can do that
var afterOpenHook func()

// lockPIDFile takes an exclusive lock on the file, which is held until
// the process exits or the file is closed
func lockPIDFile(path string) (*PIDFile, error) {
	// the file is removed by its holder when it exits, so the file that
	// was opened can be unlinked before it is locked, and then locking
	// it does not stop another process locking a new file. Check the file
	// that is locked is still the one at path, and if not try again.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
		if err != nil {
			return nil, err
		}
		if afterOpenHook != nil {
			afterOpenHook()
		}

		// another process checking for a lock, using pidFileHeld, takes a
		// brief shared lock, so only give up if it is still locked after
		// a short while
		var ok bool
		for attempt := range 3 {
			if ok, err = tryFlock(f, syscall.LOCK_EX); err != nil || ok {
				break
			}
			if attempt < 2 {
				time.Sleep(5 * time.Millisecond)
			}
		}
		if err != nil {
			f.Close()
			return nil, err
		}
		if !ok {
			info, _ := ReadPIDFile(path)
			f.Close()
			return nil, &AlreadyRunningError{Path: path, PID: info.PID}
		}

		locked, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, err
		}
		if current, err := os.Stat(path); err != nil || !os.SameFile(locked, current) {
			f.Close()
			continue
		}

		if err = f.Truncate(0); err != nil {
			f.Close()
			return nil, err
		}
		return &PIDFile{path: path, f: f}, nil
	}
	return nil, fmt.Errorf("pid file %s kept being replaced while locking it", path)
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
