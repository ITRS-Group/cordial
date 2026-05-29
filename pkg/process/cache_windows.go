package process

import (
	"errors"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/rs/zerolog/log"
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
var procCacheMap = make(map[host.Host]any)

func getProcesses[T any](h host.Host, options ...ProcessOption) (c map[int]T, ok bool) {
	procCacheMutex.Lock()
	defer procCacheMutex.Unlock()

	opts := evalProcessOptions(options...)
	refreshCache := opts.refreshCache
	if !refreshCache {
		if c, ok = procCacheMap[h].(map[int]T); ok {
			if time.Since(procCacheLastUpdate[h]) < procCacheTTL {
				return
			}
		}
	}

	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		log.Error().Err(err).Msg("failed to create toolhelp32 snapshot")
		return c, false
	}
	defer windows.CloseHandle(handle)

	c = make(map[int]T)

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err = windows.Process32First(handle, &pe); err != nil {
		log.Error().Err(err).Msg("failed to get first process")
		return c, false
	}

	for {
		if err = windows.Process32Next(handle, &pe); err != nil {
			// done
			break
		}
		if pe.ProcessID == 0 {
			continue
		}

		pid := pe.ProcessID

		var pstatus T
		if pstatus, err = ProcessStatus[T](h, int(pid), false, false); err != nil {
			if !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
				log.Error().Err(err).Msgf("failed to get process status for pid %d", pid)
			}
			continue
		}

		c[int(pid)] = pstatus
	}

	procCacheLastUpdate[h] = time.Now()
	procCacheMap[h] = c

	return c, true
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
