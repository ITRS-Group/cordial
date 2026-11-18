//go:build !linux

package cmdtest

import "os"

const stderrCaptured = false

func redirectStderrFd(*os.File) (restore func()) {
	return func() {}
}
