package process

import (
	"errors"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
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
		if pstatus, err = ProcessStatus[T](h, int(pid)); err != nil {
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

func checkAndFillCache(h host.Host, pid int, pc *ProcessInfo) {
	// check if OpenFiles is empty, if so fill it
	if len(pc.OpenFiles) == 0 {
		ProcessStatusOpenFiles(h, uint32(pid), reflect.ValueOf(pc).Elem().FieldByName("OpenFiles"))
	}

	// check if OpenSockets is zero, if so fill it
	if pc.OpenSockets == 0 {
		ProcessStatusOpenSockets(h, uint32(pid), reflect.ValueOf(pc).Elem().FieldByName("OpenSockets"))
	}

	// check if ListeningPorts is empty, if so fill it
	if pc.ListeningPorts == "" {
		ProcessStatusListeningPorts(h, uint32(pid), reflect.ValueOf(pc).Elem().FieldByName("ListeningPorts"))
	}
}
