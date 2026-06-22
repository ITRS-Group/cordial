/*
Copyright © 2023 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package process

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/itrs-group/cordial/pkg/host"
)

// ProcessInfoMinimal is an example of a structure to pass to
// instance.ProcessStatus, using a field number for `stat` and a line
// prefix for `status` tags. OpenFiles and OpenSockets fields are counts
// of their respective names.
type ProcessInfoMinimal struct {
	PID     int      `proc_pid_stat:"0" json:"-"`
	Exe     string   `json:"-"`
	Cmdline []string `json:"-"`
}

// PID returns the PID of the process started with executable name and
// all args (in any order) on host h. If not found then an err of
// os.ErrProcessDone is returned.
//
// A CustomChecker() function can be passed as an option to validate the
// args against the each process found. If the function returns true
// then the process is a match.
//
// By default a process cache is used to avoid repeated calls to the
// host to get the process entries, which can be expensive. The cache is
// updated every 5 seconds, or when the cache is empty. The cache can be
// reset/refreshed by passing the RefreshCache() option.
func PID(h host.Host, executable string, args []string, options ...ProcessOption) (pid int, err error) {
	if h == nil {
		return 0, fmt.Errorf("host cannot be nil")
	}
	if executable == "" {
		return 0, fmt.Errorf("executable must not be empty")
	}

	opts := evalProcessOptions(options...)

	c, ok := getProcesses[*ProcessInfoMinimal](h, options...)
	if !ok {
		return 0, fmt.Errorf("host %s does not support process lookups", h)
	}

	for _, pc := range c {
		// remove Linux specific suffixes
		exe := strings.TrimSuffix(pc.Exe, " (deleted)")

		if filepath.Base(exe) != executable {
			continue
		}

		if len(args) > len(pc.Cmdline)-1 {
			// not enough arguments to test, so it can't match all args
			continue
		}

		if opts.checkFunc != nil {
			if opts.checkFunc(opts.checkArg, pc.Cmdline) {
				return pc.PID, nil
			}
			continue
		}

		if len(args) == 0 {
			// no args to check, so return the first match
			return pc.PID, nil
		}

		// check that all args passed to the function are somewhere on
		// the command line. we do not check the order of the args, just
		// that they are all present in the command line
		if slices.ContainsFunc(pc.Cmdline[1:], func(a string) bool {
			return slices.Contains(args, a)
		}) {
			return pc.PID, nil
		}
	}

	return 0, os.ErrProcessDone
}

// GetProcessInfo returns information about the process pid on host h.
func GetProcessInfo[T any](h host.Host, pid int, options ...ProcessOption) (pi T, err error) {
	opts := evalProcessOptions(options...)

	p := any(pi).(T)

	cache, ok := getProcesses[T](h, options...)
	if !ok {
		return pi, fmt.Errorf("host %s does not support process lookups", h)
	}
	if p, ok = cache[pid]; ok {
		if opts.fetchLazy {
			// check and fill cache for lazy (expensive) fields
			checkAndFillCache(h, pid, p)
		}
		return any(p).(T), nil
	}

	return pi, os.ErrProcessDone
}

// ClearCache clears the process cache for host h. This can be used to
// force a refresh of the cache on the next call to PID() or
// GetProcessInfo(). This is useful if processes are expected to have
// started or stopped since the last cache update, and an up to date
// check is required. Note that the cache is automatically refreshed
// every 5 seconds, so this function is only needed if an immediate
// refresh is required.
func ClearCache() {
	clearCache()
}
