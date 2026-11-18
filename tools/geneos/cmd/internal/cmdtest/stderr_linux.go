package cmdtest

import (
	"os"
	"syscall"
)

const stderrCaptured = true

// redirectStderrFd makes file descriptor 2 refer to w until the
// returned function is called, as some packages keep a reference to
// the original os.Stderr
func redirectStderrFd(w *os.File) (restore func()) {
	saved, err := syscall.Dup(2)
	if err != nil {
		return func() {}
	}
	if err = syscall.Dup3(int(w.Fd()), 2, 0); err != nil {
		syscall.Close(saved)
		return func() {}
	}
	return func() {
		syscall.Dup3(saved, 2, 0)
		syscall.Close(saved)
	}
}
