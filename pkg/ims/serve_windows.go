//go:build windows

package ims

import "os"

// reloadSignals are the signals that make the server reload, which there
// are none of on Windows
func reloadSignals() []os.Signal { return nil }
