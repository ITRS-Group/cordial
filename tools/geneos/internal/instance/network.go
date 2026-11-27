/*
Copyright © 2022 ITRS Group

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

package instance

import (
	"errors"
	"log/slog"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// GetAllPorts gets all used ports in config files on a specific remote
// and also all listening ports on the same host. Returns a map of port
// to struct{} for each lookup.
func GetAllPorts(h *geneos.Host) (ports map[uint16]struct{}) {
	if h == geneos.ALL {
		log.Error("GetAllPorts() called with all hosts")
		os.Exit(1)
	}
	ports = make(map[uint16]struct{})
	if h.IsUnknown() {
		return
	}
	for _, c := range Instances(h, nil) {
		if c.Loaded().IsZero() {
			c.Log().Error("cannot load configuration for instance")
			continue
		}
		if port := config.Get[int](c.Config(), "port"); port != 0 {
			ports[uint16(port)] = struct{}{}
		}
	}

	// add all listening ports
	listening := make(map[int]int)
	if err := process.AllTCPListenPorts(h, listening); err != nil {
		return
	}
	for _, v := range listening {
		ports[uint16(v)] = struct{}{}
	}
	return
}

// CheckPort checks if the given port is in use by the specified instance.
// Returns true if the port is in use, false otherwise. Returns an error
// if the instance's PID cannot be determined.
func CheckPort(i geneos.Instance, port uint16) (inUse bool, err error) {
	pid, err := GetLivePID(i)
	if err != nil {
		return
	}
	ports := process.ListeningPorts(i.Host(), pid)
	inUse = slices.Contains(ports, int(port))
	return
}

// PortInUse checks if the given port is in use on the given host.
func PortInUse(h *geneos.Host, port uint16) (inUse bool, err error) {
	if h == geneos.ALL {
		err = errors.New("PortInUse() called with all hosts")
		return
	}
	ports := GetAllPorts(h)
	_, inUse = ports[port]
	return
}

// ByPort returns an instance on host h that is configured to use port.
// Returns an error if no instance is found or if the configuration
// cannot be loaded for any instance on the host. Will not check if the
// port is actually in use by any process.
func ByPort(h *geneos.Host, port uint16) (i geneos.Instance, err error) {
	if h == geneos.ALL {
		err = errors.New("ByPort() called with all hosts")
		return
	}

	for _, c := range Instances(h, nil) {
		if c.Loaded().IsZero() {
			c.Log().Error("cannot load configuration for instance")
			continue
		}
		if p := config.Get[uint16](c.Config(), "port"); p != 0 {
			if p == port {
				i = c
				return
			}
		}
	}
	err = geneos.ErrNotExist
	return
}

// NextFreePort returns the next available (unallocated and unused) TCP
// listening port for component ct on host h.
//
// The range of ports available for a component is defined in the
// configuration for the user and for each component type. A port is
// available if it is neither allocated to any other instance on the
// same host (of any component type) and also is not in use by any other
// process which may not be a Geneos instance.
//
// Each range is a comma separated list of single port number, e.g.
// "7036", a min-max inclusive range, e.g. "7036-8036" or a 'start-'
// open ended range, e.g. "7041-". Ranges can also be denoted by
// double-dot in addition to single dashes '-'.
//
// some limits based on
// https://en.wikipedia.org/wiki/List_of_TCP_and_UDP_port_numbers
//
// not concurrency safe at this time
func NextFreePort(h *geneos.Host, ct *geneos.Component) uint16 {
	from := ct.GlobalSetting(ct.PortRange)
	log.Debug("looking for next free port", slog.String("portRange", ct.PortRange), slog.String("ranges", from))
	return firstFreePort(from, GetAllPorts(h))
}

// FreePortFrom returns the first port at or above start, and not below
// 1024, that is neither allocated to an instance on host h nor in use by
// any process. It ignores the port ranges of the components and is the
// last resort when none of those has a free port. It returns 0 only if
// every port up to 65535 is used.
func FreePortFrom(h *geneos.Host, start uint16) uint16 {
	used := GetAllPorts(h)
	// use an int to loop to avoid wrapping at the top of the range
	for p := max(int(start), 1024); p <= math.MaxUint16; p++ {
		if _, ok := used[uint16(p)]; !ok {
			return uint16(p)
		}
	}
	return 0
}

// firstFreePort returns the first port in the comma separated list of
// ports and ranges in ranges that is not in used, or 0 if there is
// none. Entries can have surrounding white space, and invalid entries,
// those that are not numbers or are outside 1-65535, are ignored.
//
// Open-ended ranges, e.g. "7100-", end at 49151, the top of the
// registered ports, unless they start above that in which case they end
// at 65535.
func firstFreePort(ranges string, used map[uint16]struct{}) uint16 {
	for entry := range strings.SplitSeq(ranges, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// split on dash or ".."
		start, end, isRange := strings.Cut(entry, "-")
		if !isRange {
			start, end, isRange = strings.Cut(entry, "..")
		}

		first, err := strconv.Atoi(strings.TrimSpace(start))
		if err != nil || first < 1 || first > math.MaxUint16 {
			continue
		}

		last := first
		if isRange {
			if end = strings.TrimSpace(end); end == "" {
				last = 49151
				if first > last {
					last = math.MaxUint16
				}
			} else if last, err = strconv.Atoi(end); err != nil {
				continue
			}
			last = min(last, math.MaxUint16)
			if last < first {
				continue
			}
		}

		// use an int to loop to avoid wrapping at the top of the range
		for p := first; p <= last; p++ {
			if _, ok := used[uint16(p)]; !ok {
				return uint16(p)
			}
		}
	}
	return 0
}
