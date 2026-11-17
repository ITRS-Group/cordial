//go:build !windows

package process

import (
	"errors"
	"os"
	"syscall"
)

// Signals used to control a process
var (
	// SigTerm asks a process to stop, gracefully
	SigTerm os.Signal = syscall.SIGTERM
	// SigKill stops a process immediately
	SigKill os.Signal = syscall.SIGKILL
	// SigReload asks a process to reload its configuration
	SigReload os.Signal = syscall.SIGUSR1
)

// processAlive reports if there is a process with the PID
func processAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}
