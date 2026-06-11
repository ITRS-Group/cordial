//go:build !windows

package instance

import (
	"time"

	"github.com/itrs-group/cordial/pkg/process"
)

// ProcessInfo is an example of a structure to pass to
// instance.ProcessStatus, using a field number for `stat` and a line
// prefix for `status` tags. OpenFiles and OpenSockets fields are counts
// of their respective names. Some fields may be expensive to fill for
// all processes, so they are marked with `cache:"lazy"` to indicate
// that they should be filled on demand when requested, rather than when
// the process information is first retrieved.
type ProcessInfo struct {
	PID             int           `proc_pid_stat:"0" json:"-"`
	PPID            int           `proc_pid_stat:"3"`
	Utime           time.Duration `proc_pid_stat:"13"`
	Stime           time.Duration `proc_pid_stat:"14"`
	CUtime          time.Duration `proc_pid_stat:"15"`
	CStime          time.Duration `proc_pid_stat:"16"`
	UIDs            []string      `proc_pid_status:"Uid" json:"-"`
	GIDs            []string      `proc_pid_status:"Gid" json:"-"`
	State           string        `proc_pid_status:"State"`
	Threads         int64         `proc_pid_status:"Threads"`
	VmRSS           int64         `proc_pid_status:"VmRSS"`
	VmHWM           int64         `proc_pid_status:"VmHWM"`
	RssAnon         int64         `proc_pid_status:"RssAnon"`
	RssFile         int64         `proc_pid_status:"RssFile"`
	RssShmem        int64         `proc_pid_status:"RssShmem"`
	CpusAllowedList string        `proc_pid_status:"Cpus_allowed_list"`

	// special fields that are not from /proc/PID/stat or
	// /proc/PID/status but are calculated from other information, such
	// as the number of open files and sockets
	//
	// filling in these fields for all processes can be expensive, so
	// they are marked with `cache:"lazy"` to indicate that they should
	// be filled on demand when requested, rather than when the process
	// information is first retrieved.
	OpenFiles      []process.ProcessFDs `cache:"lazy" json:"-"` // calculated from /proc/PID/fd
	OpenSockets    int64                `cache:"lazy" json:"-"` // calculated from /proc/PID/fd and /proc/PID/net/tcp and /proc/PID/net/udp
	ListeningPorts string               `cache:"lazy" json:"-"` // calculated from /proc/PID/net/tcp and /proc/PID/net/udp

	Cwd       string    `json:"-"` // calculated from /proc/PID/cwd
	Exe       string    `json:"-"`
	Cmdline   []string  `json:"-"`
	StartTime time.Time `json:"-"`
	Children  []int     `json:"-"`
	UID       int       `json:"-"` // real UID
	EUID      int       `json:"-"` // effective UID
	GID       int       `json:"-"` // real GID
	EGID      int       `json:"-"` // effective GID
	Username  string    `json:"-"`
	Groupname string    `json:"-"`
}
