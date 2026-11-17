//go:build windows

package process

import (
	"os"

	"golang.org/x/sys/windows"
)

// Signals used to control a process. Windows has no signals other than
// to kill a process, so stopping is immediate and there is no reload.
var (
	// SigTerm stops a process, immediately on Windows
	SigTerm os.Signal = os.Kill
	// SigKill stops a process immediately
	SigKill os.Signal = os.Kill
	// SigReload is nil as it is not supported on Windows
	SigReload os.Signal
)

// processAlive reports if there is a running process with the PID
func processAlive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)

	var code uint32
	if err = windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == 259 // STILL_ACTIVE
}
