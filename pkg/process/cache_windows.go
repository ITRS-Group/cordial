package process

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/itrs-group/cordial/pkg/host"
)

// procCache is a map of host to procCache, which is used to cache
// process entries for each host. It is used to avoid repeated calls to
// the host to get the process entries, which can be expensive.
//
// The cache is marked stale after 5 seconds, or when the cache is
// empty.
var procCacheTTL = 5 * time.Second
var procCacheMutex sync.Mutex
var procCacheLastUpdate = make(map[host.Host]time.Time)

// var procCacheMap = make(map[host.Host]any)
var procCacheMap = make(map[host.Host]map[string]any) // host / type / cache

func getProcesses[T any](h host.Host, options ...ProcessOption) (c map[int]T, ok bool) {
	procCacheMutex.Lock()
	defer procCacheMutex.Unlock()

	tn := fmt.Sprintf("%T", (*T)(nil))

	opts := evalProcessOptions(options...)
	refreshCache := opts.refreshCache
	if !refreshCache {
		if c, ok = procCacheMap[h][tn].(map[int]T); ok {
			if time.Since(procCacheLastUpdate[h]) < procCacheTTL {
				return
			}
		}
	}

	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		err = fmt.Errorf("failed to create toolhelp32 snapshot: %w", err)
		return c, false
	}
	defer windows.CloseHandle(handle)

	c = make(map[int]T)

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err = windows.Process32First(handle, &pe); err != nil {
		err = fmt.Errorf("failed to get first process: %w", err)
		return c, false
	}

	for {
		if pe.ProcessID != 0 {
			pid := pe.ProcessID
			var pstatus T
			if pstatus, err = ProcessStatus[T](h, int(pid), false, false); err != nil {
				if !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
					err = fmt.Errorf("failed to get process status for pid %d: %w", pid, err)
				}
			} else {
				c[int(pid)] = pstatus
			}
		}
		if err = windows.Process32Next(handle, &pe); err != nil {
			break
		}
	}

	st := reflect.TypeOf((*T)(nil)).Elem()
	if st.Kind() == reflect.Pointer {
		st = st.Elem()
	}
	if _, hasPID := st.FieldByName("PID"); hasPID {
		if _, hasPPID := st.FieldByName("PPID"); hasPPID {
			if _, hasChildren := st.FieldByName("Children"); hasChildren {
				for _, child := range c {
					childValue := reflect.ValueOf(child)
					if childValue.Kind() == reflect.Pointer {
						childValue = childValue.Elem()
					}
					pidField := childValue.FieldByName("PID")
					ppidField := childValue.FieldByName("PPID")
					if !pidField.IsValid() || !ppidField.IsValid() || pidField.Kind() < reflect.Int || pidField.Kind() > reflect.Int64 || ppidField.Kind() < reflect.Int || ppidField.Kind() > reflect.Int64 {
						continue
					}
					parent, ok := c[int(ppidField.Int())]
					if !ok {
						continue
					}
					parentValue := reflect.ValueOf(parent)
					if parentValue.Kind() == reflect.Pointer {
						parentValue = parentValue.Elem()
					}
					children := parentValue.FieldByName("Children")
					if children.IsValid() && children.CanSet() && children.Kind() == reflect.Slice && children.Type().Elem().Kind() == reflect.Int {
						children.Set(reflect.Append(children, reflect.ValueOf(int(pidField.Int()))))
						c[int(ppidField.Int())] = parent
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
