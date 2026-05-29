//go:build windows

package instance

import (
	"time"

	"github.com/itrs-group/cordial/pkg/process"
)

type ProcessInfo struct {
	PID  int
	PPID int
	UID  int
	GID  int

	OpenFiles      []process.ProcessFDs
	OpenSockets    int64
	ListeningPorts string

	Cwd       string
	Exe       string
	Cmdline   []string
	StartTime time.Time
	Children  []int

	Username  string
	Groupname string
}
