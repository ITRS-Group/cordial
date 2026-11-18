package process

import (
	"reflect"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/host"
)

// procCaches holds the cached process entries for each host, which are
// used to avoid repeated calls to the host to get the process entries,
// which can be expensive.
//
// The entries for each type are marked stale after procCacheTTL, or when
// the cache is empty.

// procCacheMutex protects the map of hosts only. Each host has its own
// lock, so that a slow scan of one host does not hold up the others.
var procCacheMutex sync.Mutex
var procCaches = make(map[host.Host]*hostCache)

// hostCache is the cache for one host, by type of the entries
type hostCache struct {
	mu    sync.Mutex
	types map[reflect.Type]*typeCache
}

// typeCache is the entries of one type, which is a map[int]T
type typeCache struct {
	entries any
	updated time.Time
}

func cacheFor(h host.Host) *hostCache {
	procCacheMutex.Lock()
	defer procCacheMutex.Unlock()
	hc := procCaches[h]
	if hc == nil {
		hc = &hostCache{types: make(map[reflect.Type]*typeCache)}
		procCaches[h] = hc
	}
	return hc
}

// getProcesses returns the process entries for host h from the cache, if
// the cache is not stale. If the cache is stale or empty, it updates the
// cache and returns the new entries. The type of the entries is determined
// by the type parameter T, which must be a pointer to a struct that can be
// populated from the process status information.
//
// The function returns the process entries and a boolean indicating whether
// the cache was successfully updated or not. If the host does not support
// process lookups, it returns false.
func getProcesses[T any](h host.Host, options ...ProcessOption) (c map[int]T, ok bool) {
	if h == nil {
		return nil, false
	}
	hc := cacheFor(h)
	hc.mu.Lock()
	defer hc.mu.Unlock()
	return loadProcesses[T](hc, h, evalProcessOptions(options...))
}

// loadProcesses is getProcesses for a host with its cache locked, which
// is also held by the caller while using entries
func loadProcesses[T any](hc *hostCache, h host.Host, opts *processOptions) (c map[int]T, ok bool) {
	tt := reflect.TypeFor[T]()
	if tt.Kind() != reflect.Pointer || tt.Elem().Kind() != reflect.Struct {
		return nil, false
	}

	if !opts.refreshCache {
		if tc := hc.types[tt]; tc != nil {
			if c, ok = tc.entries.(map[int]T); ok && time.Since(tc.updated) < procCacheTTL {
				return
			}
		}
	}

	// cache is empty or expired, update it
	if c, ok = scanProcesses[T](h); !ok {
		return c, false
	}

	hc.types[tt] = &typeCache{entries: c, updated: time.Now()}
	return c, true
}

func clearCache() {
	procCacheMutex.Lock()
	defer procCacheMutex.Unlock()
	procCaches = make(map[host.Host]*hostCache)
}

// checkAndFillCache fills any lazy fields in the cached entry pc. The
// cache for the host must be locked by the caller.
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

// copyEntry returns a shallow copy of the struct that pc points to, so
// that callers do not share the cached entry
func copyEntry[T any](pc T) T {
	pv := reflect.ValueOf(pc)
	if pv.Kind() != reflect.Pointer || pv.IsNil() || pv.Elem().Kind() != reflect.Struct {
		return pc
	}
	n := reflect.New(pv.Elem().Type())
	n.Elem().Set(pv.Elem())
	return n.Interface().(T)
}
