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
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// GetAllPorts gets all used ports in config files on a specific remote
// and also all listening ports on the same host. Returns a map of port
// to bool "true" for each lookup.
func GetAllPorts(h *geneos.Host) (ports map[uint16]bool) {
	if h == geneos.ALL {
		log.Fatal().Msg("getports() called with all hosts")
	}
	ports = make(map[uint16]bool)
	for _, c := range Instances(h, nil) {
		if c.Loaded().IsZero() {
			log.Error().Msgf("cannot load configuration for %s", c)
			continue
		}
		if port := config.Get[int](c.Config(), c.Config().Join("port")); port != 0 {
			ports[uint16(port)] = true
		}
	}

	// add all listening ports
	listening := make(map[int]int)
	if err := process.AllTCPListenPorts(h, listening); err != nil {
		return
	}
	for _, v := range listening {
		ports[uint16(v)] = true
	}
	return
}

// ByPort returns an instance on host h that is configured to use port.
// Returns an error if no instance is found or if the configuration
// cannot be loaded for any instance on the host. Will not check if the
// port is actually in use by any process.
func ByPort(h *geneos.Host, port uint16) (i geneos.Instance, err error) {
	if h == geneos.ALL {
		err = errors.New("getports() called with all hosts")
		return
	}

	for _, c := range Instances(h, nil) {
		if c.Loaded().IsZero() {
			log.Error().Msgf("cannot load configuration for %s", c)
			continue
		}
		if p := config.Get[uint16](c.Config(), c.Config().Join("port")); p != 0 {
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
	log.Debug().Msgf("looking for %s, default %s", ct.PortRange, ct.ConfigAliases[ct.PortRange])
	from := config.Get[string](config.Global(), ct.PortRange, config.DefaultValue(ct.ConfigAliases[ct.PortRange]))
	used := GetAllPorts(h)
	for p := range strings.SplitSeq(from, ",") {
		// split on dash or ".."
		m := strings.SplitN(p, "-", 2)
		if len(m) == 1 {
			m = strings.SplitN(p, "..", 2)
		}

		if len(m) > 1 {
			var min uint16
			mn, err := strconv.Atoi(m[0])
			if err != nil {
				continue
			}
			if mn < 0 || mn > 65534 {
				min = 65535
			} else {
				min = uint16(mn)
			}
			if m[1] == "" {
				m[1] = "49151"
			}
			max, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}
			if int(min) >= max {
				continue
			}
			for i := min; int(i) <= max; i++ {
				if _, ok := used[i]; !ok {
					// found an unused port
					return i
				}
			}
		} else {
			var p1 uint16
			p, err := strconv.Atoi(m[0])
			if err != nil {
				continue
			}
			if p < 0 || p > 65534 {
				p1 = 65535
			} else {
				p1 = uint16(p)
			}
			if _, ok := used[p1]; !ok {
				return p1
			}
		}
	}
	return 0
}
