//go:build !windows

package process

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/pkg/sftp"
	"golang.org/x/sys/unix"
)

// procCache is a map of host to procCache, which is used to cache
// process entries for each host. It is used to avoid repeated calls to
// the host to get the process entries, which can be expensive.
//
// The cache is marked stale after 30 seconds, or when the cache is
// empty.
var procCacheTTL = 30 * time.Second
var procCacheMutex sync.Mutex
var procCacheLastUpdate = make(map[host.Host]time.Time)
var procCacheMap = make(map[host.Host]map[string]any) // host / type / cache

// getProcesses returns the process entries for host h from the cache, if
// the cache is not stale. If the cache is stale or empty, it updates the
// cache and returns the new entries. The type of the entries is determined
// by the type parameter T, which must be a struct that can be populated
// from the process status information.
//
// The cache is updated by reading the /proc filesystem on the host, and
// building a map of PID to process status information. If the type T has
// PID, PPID and Children fields, it also builds child lists based on linked
// PID and PPID fields.
//
// The function returns the process entries and a boolean indicating whether
// the cache was successfully updated or not. If the host does not support
// process lookups, it returns false.
func getProcesses[T any](h host.Host, options ...ProcessOption) (c map[int]T, ok bool) {
	procCacheMutex.Lock()
	defer procCacheMutex.Unlock()

	opts := evalProcessOptions(options...)
	refreshCache := opts.refreshCache

	tn := fmt.Sprintf("%T", (*T)(nil))

	if !refreshCache {
		if c, ok = procCacheMap[h][tn].(map[int]T); ok {
			if time.Since(procCacheLastUpdate[h]) < procCacheTTL {
				return
			}
		}
	}

	// cache is empty or expired, update it
	dirs, err := h.Glob("/proc/[0-9]*")
	if err != nil {
		return c, false
	}
	c = make(map[int]T, len(dirs))

	// if the type T has a PID, PPID and Children fields, build child
	// lists based on linked PID and PPID fields. This is a bit hacky
	// but it avoids having to define a separate struct for the cache
	// that includes these fields.
	st := reflect.TypeFor[T]().Elem()

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

		// set starttime is a field exists and is a time.Time
		pt := reflect.TypeFor[T]().Elem()
		if _, ok := pt.FieldByName("StartTime"); ok {
			pv := reflect.ValueOf(pstatus)
			if pv.Kind() == reflect.Pointer {
				pv = pv.Elem()
			}
			fv := pv.FieldByName("StartTime")
			if fv.Type() == reflect.TypeFor[time.Time]() ||
				(fv.Kind() == reflect.Pointer && fv.Type().Elem() == reflect.TypeFor[time.Time]()) {
				fv.Set(reflect.ValueOf(st.ModTime()))
			}
		}

		c[pid] = pstatus
	}

	if _, ok := st.FieldByName("PID"); ok {
		if _, ok := st.FieldByName("PPID"); ok {
			if _, ok := st.FieldByName("Children"); ok {
				for _, p := range c {
					pv := reflect.ValueOf(p)
					if pv.Kind() == reflect.Pointer {
						pv = pv.Elem()
					}
					pidField := pv.FieldByName("PID")
					ppidField := pv.FieldByName("PPID")
					childrenField := pv.FieldByName("Children")

					if pidField.IsValid() && ppidField.IsValid() && childrenField.IsValid() {
						pid := int(pidField.Int())
						ppid := int(ppidField.Int())

						if parent, ok := c[ppid]; ok {
							ppv := reflect.ValueOf(parent)
							if ppv.Kind() == reflect.Pointer {
								ppv = ppv.Elem()
							}
							parentChildrenField := ppv.FieldByName("Children")
							if parentChildrenField.IsValid() && parentChildrenField.Kind() == reflect.Slice && parentChildrenField.Type().Elem().Kind() == reflect.Int {
								parentChildrenField.Set(reflect.Append(parentChildrenField, reflect.ValueOf(pid)))
								c[ppid] = parent
							}
						}
					}
				}
			}
		}
	}

	procCacheLastUpdate[h] = time.Now()
	if procCacheMap[h] == nil {
		procCacheMap[h] = make(map[string]any)
	}
	procCacheMap[h][tn] = c

	return c, true
}

func clearCache() {
	procCacheMap = make(map[host.Host]map[string]any)
	procCacheLastUpdate = make(map[host.Host]time.Time)
}

func checkAndFillCache[T any](h host.Host, pid int, pc T) {
	// check if OpenFiles is empty, if so fill it
	sv := reflect.ValueOf(pc).Elem()
	if sv.FieldByName("OpenFiles").IsValid() && sv.FieldByName("OpenFiles").IsZero() {
		ProcessStatusOpenFiles(h, pid, sv.FieldByName("OpenFiles"))
	}

	// check if OpenSockets is zero, if so fill it
	if sv.FieldByName("OpenSockets").IsValid() && sv.FieldByName("OpenSockets").IsZero() {
		ProcessStatusOpenSockets(h, pid, sv.FieldByName("OpenSockets"))
	}

	// check if ListeningPorts is empty, if so fill it
	if sv.FieldByName("ListeningPorts").IsValid() && sv.FieldByName("ListeningPorts").IsZero() {
		ProcessStatusListeningPorts(h, pid, sv.FieldByName("ListeningPorts"))
	}
}
