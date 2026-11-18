/*
Copyright © 2026 ITRS Group

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

// Package tlsports implements the conventional switching of a
// component's listening port when its TLS capability changes.
//
// Only the well-known pairs of insecure and secure ports are ever
// switched and only when the instance is configured with one of the
// pair and its TLS state disagrees with it. Every other port is
// user-defined and is always preserved.
package tlsports

import (
	"log/slog"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// Pair is the conventional insecure and secure port numbers for a
// component.
type Pair struct {
	Insecure uint16
	Secure   uint16
}

var (
	// Gateway is the pair of ports for Gateways, 7039 for insecure and
	// 7038 for TLS connections.
	Gateway = []Pair{{Insecure: 7039, Secure: 7038}}

	// Webserver is the pair of ports for Web Dashboard servers, 8080
	// for HTTP and 8443 for HTTPS, and the limited case of the standard
	// 80 and 443.
	Webserver = []Pair{{Insecure: 8080, Secure: 8443}, {Insecure: 80, Secure: 443}}
)

// Target returns the port to use instead of port for a given TLS
// state. ok is only true if port is one of the pairs and is the member
// that does not match secure.
func Target(pairs []Pair, port uint16, secure bool) (target uint16, ok bool) {
	for _, p := range pairs {
		if secure && port == p.Insecure {
			return p.Secure, true
		}
		if !secure && port == p.Secure {
			return p.Insecure, true
		}
	}
	return 0, false
}

// Apply updates the "port" of instance i in-memory, if the instance's
// TLS capability does not match the member of a pair that it is
// configured to use and the other member is free. It returns true if
// the port was changed and the caller is responsible for saving the
// configuration.
//
// If the other member of the pair is in use by another instance or
// process then the configured port is kept and a warning logged, it
// never moves to any other port.
func Apply(i geneos.Instance, pairs []Pair) (changed bool) {
	cf := i.Config()
	secure := instance.IsTLSCapable(i)
	port := config.Get[uint16](cf, "port")

	target, ok := Target(pairs, port, secure)
	if !ok {
		return false
	}

	// do not move if this instance is already listening on the other port
	if inUse, _ := instance.CheckPort(i, target); inUse {
		return false
	}

	if _, used := instance.GetAllPorts(i.Host())[target]; used {
		i.Log().Warn("port conventionally used for the TLS configuration is already in use, keeping configured port",
			slog.Int("port", int(port)), slog.Int("preferred", int(target)), slog.Bool("tls", secure))
		return false
	}

	i.Log().Debug("switching port to match TLS configuration",
		slog.Int("from", int(port)), slog.Int("to", int(target)), slog.Bool("tls", secure))
	config.Set(cf, "port", target)
	return true
}
