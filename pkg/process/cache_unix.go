//go:build !windows

package process

import (
	"reflect"
	"strconv"
	"time"

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/pkg/sftp"
	"golang.org/x/sys/unix"
)

// procCacheTTL is how long the cached process entries are used for
var procCacheTTL = 30 * time.Second

// scanProcesses reads the /proc filesystem on the host, and builds a map
// of PID to process status information. If the type T has PID, PPID and
// Children fields, it also builds child lists based on linked PID and
// PPID fields.
func scanProcesses[T any](h host.Host) (c map[int]T, ok bool) {
	tt := reflect.TypeFor[T]()

	dirs, err := h.Glob("/proc/[0-9]*")
	if err != nil {
		return c, false
	}
	c = make(map[int]T, len(dirs))

	st := tt.Elem()

	// check which proc files we need to read
	var getStat, getStatus bool
	for field := range st.Fields() {
		if _, ok := field.Tag.Lookup("proc_pid_stat"); ok {
			getStat = true
		}
		if _, ok := field.Tag.Lookup("proc_pid_status"); ok {
			getStatus = true
		}
	}

	var myUID, myGID uint32 = ^uint32(0), ^uint32(0)

	if !h.IsLocalhost() {
		info, err := h.Stat("/proc/self") // prime the connection to avoid timing out on the first real stat
		if err == nil {
			switch sys := info.Sys().(type) {
			case *sftp.FileStat:
				myUID = sys.UID
				myGID = sys.GID
			case *unix.Stat_t:
				myUID = sys.Uid
				myGID = sys.Gid
			default:
				// log.Debug().Msgf("host %s uses unknown method for process lookups, stat of /proc/self: %+v", h.String(), sys)
			}
		}
	}

	_, hasStartTime := st.FieldByName("StartTime")

	for _, dir := range dirs {
		st, err := h.Stat(dir)
		if err != nil {
			continue
		}
		if !st.IsDir() {
			continue
		}
		if !h.IsLocalhost() {
			switch sys := st.Sys().(type) {
			case *sftp.FileStat:
				if sys.UID != myUID || sys.GID != myGID {
					continue
				}
			case *unix.Stat_t:
				if sys.Uid != myUID || sys.Gid != myGID {
					continue
				}
			default:
				// log.Debug().Msgf("host %s uses unknown method for process lookups, stat of %s: %+v", h.String(), dir, sys)
			}
		}

		pid, err := strconv.Atoi(st.Name())
		if err != nil {
			continue
		}

		var pstatus T
		if pstatus, err = processStatus[T](h, pid, getStat, getStatus); err != nil {
			continue
		}

		if hasStartTime {
			setStartTime(reflect.ValueOf(pstatus), st.ModTime())
		}

		c[pid] = pstatus
	}

	// if the type T has a PID, PPID and Children fields, build child
	// lists based on linked PID and PPID fields. This is a bit hacky
	// but it avoids having to define a separate struct for the cache
	// that includes these fields.
	if _, ok := st.FieldByName("PID"); ok {
		if _, ok := st.FieldByName("PPID"); ok {
			if _, ok := st.FieldByName("Children"); ok {
				for _, p := range c {
					pv := reflect.ValueOf(p).Elem()
					pidField := pv.FieldByName("PID")
					ppidField := pv.FieldByName("PPID")
					childrenField := pv.FieldByName("Children")

					if pidField.IsValid() && ppidField.IsValid() && childrenField.IsValid() {
						pid := int(pidField.Int())
						ppid := int(ppidField.Int())

						if parent, ok := c[ppid]; ok {
							parentChildrenField := reflect.ValueOf(parent).Elem().FieldByName("Children")
							if parentChildrenField.IsValid() && parentChildrenField.Kind() == reflect.Slice && parentChildrenField.Type().Elem().Kind() == reflect.Int {
								parentChildrenField.Set(reflect.Append(parentChildrenField, reflect.ValueOf(pid)))
							}
						}
					}
				}
			}
		}
	}

	return c, true
}

// setStartTime sets the StartTime field of the struct that pv points to,
// if there is one that is a time.Time or *time.Time
func setStartTime(pv reflect.Value, t time.Time) {
	if pv.Kind() == reflect.Pointer {
		pv = pv.Elem()
	}
	fv := pv.FieldByName("StartTime")
	if !fv.IsValid() || !fv.CanSet() {
		return
	}
	switch {
	case fv.Type() == reflect.TypeFor[time.Time]():
		fv.Set(reflect.ValueOf(t))
	case fv.Type() == reflect.TypeFor[*time.Time]():
		fv.Set(reflect.ValueOf(new(t)))
	}
}
