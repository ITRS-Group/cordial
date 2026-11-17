//go:build !windows

package ims

import (
	"os"

	"github.com/itrs-group/cordial/pkg/process"
)

// reloadSignals are the signals that make the server reload
func reloadSignals() []os.Signal { return []os.Signal{process.SigReload} }
