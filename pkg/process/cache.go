package process

import (
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/rs/zerolog/log"
)

type procCache struct {
	// LastUpdate is the time when the cache was last updated
	LastUpdate time.Time

	// Entries is the map of process entries, indexed by PID
	Entries map[int]*ProcessInfo
}

// procCache is a map of host to procCache, which is used to cache
// process entries for each host. It is used to avoid repeated calls to
// the host to get the process entries, which can be expensive.
//
// The cache is marked stale after 5 seconds, or when the cache is
// empty.
var procCacheTTL = 5 * time.Second
var procCacheMutex sync.Mutex
var procCacheLastUpdate = make(map[host.Host]time.Time)
var procCacheMap = make(map[host.Host]any)

func getProcesses[T any](h host.Host, refreshCache bool) (c map[int]T, ok bool) {
	procCacheMutex.Lock()
	defer procCacheMutex.Unlock()

	if !refreshCache {
		if c, ok = procCacheMap[h].(map[int]T); ok {
			if time.Since(procCacheLastUpdate[h]) < procCacheTTL {
				return
			}
		}
		// if not found, or the type has changed, refresh the cache,
		// drop through
	}

	// cache is empty or expired, update it
	dirs, err := h.Glob("/proc/[0-9]*")
	if err != nil {
		return c, false
	}
	c = make(map[int]T, len(dirs))

	for _, dir := range dirs {
		st, err := h.Stat(dir)
		if err != nil {
			log.Debug().Err(err).Msgf("failed to stat %s", dir)
			continue
		}
		if !st.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(st.Name())
		if err != nil {
			log.Debug().Err(err).Msgf("failed to parse pid from %s", dir)
			continue
		}

		var pstatus T
		if pstatus, err = ProcessStatus[T](h, pid); err != nil {
			log.Debug().Err(err).Msgf("failed to get process status for pid %d", pid)
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

	// if the type T has a PID, PPID and Children fields, build child
	// lists based on linked PID and PPID fields. This is a bit hacky
	// but it avoids having to define a separate struct for the cache
	// that includes these fields.
	st := reflect.TypeFor[T]().Elem()

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
	procCacheMap[h] = c

	return c, true
}
