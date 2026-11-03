//go:build windows

package instance

import (
	"time"

	"github.com/itrs-group/cordial/pkg/process"
)

// ProcessInfo TODO: Describe its purpose and intended use.
type ProcessInfo struct {
	PID    int
	PPID   int
	UID    int
	GID    int
	Utime  time.Duration
	Stime  time.Duration
	CUtime time.Duration
	CStime time.Duration

	State           string
	Threads         int64
	VmRSS           int64
	VmHWM           int64
	RssAnon         int64
	RssFile         int64
	RssShmem        int64
	CpusAllowedList string

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
