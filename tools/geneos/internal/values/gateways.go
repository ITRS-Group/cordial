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

package values

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
)

// The conventional Gateway listening ports. A gateway entry in a SAN or
// floating probe that uses one of these ports is moved to the other
// when the probe's TLS capability changes, all other ports are always
// preserved.
const (
	GatewayInsecurePort = "7039"
	GatewaySecurePort   = "7038"
)

// gateway - name:port
type Gateways []string

// GatewaysOptionstext TODO: Describe its purpose and intended use.
const GatewaysOptionstext = "A gateway connection in the format HOSTNAME:PORT\n(Repeat as required, san and floating only)"

// String returns the gateways as a comma-separated list.
func (i *Gateways) String() string {
	if i == nil {
		return ""
	}
	return strings.Join(*i, ",")
}

// Set TODO: Document its behavior, inputs, and effects.
func (i *Gateways) Set(value string) error {
	if *i == nil {
		*i = Gateways{}
	}
	*i = append(*i, value)
	return nil
}

// Type TODO: Document its behavior, inputs, and effects.
func (i *Gateways) Type() string {
	return "HOSTNAME:PORT"
}

// gatewayHost returns the host part of a "host[:port]" gateway entry.
func gatewayHost(entry string) string {
	host, _, _ := cutGatewayPort(entry)
	return host
}

// cutGatewayPort splits a gateway entry at the last colon. An entry
// without a colon is all host.
func cutGatewayPort(entry string) (host, port string, found bool) {
	if idx := strings.LastIndex(entry, ":"); idx >= 0 {
		return entry[:idx], entry[idx+1:], true
	}
	return entry, "", false
}

// GatewaysFromConfig returns the "gateways" setting of a SAN or
// floating instance as a slice of "host:port" strings.
//
// The current storage form is a list of strings, but older versions of
// the configuration used a map of host name to port, which is accepted
// and converted (in sorted host order) with legacy set to true so that
// callers can write back the migrated form.
func GatewaysFromConfig(cf *config.Config) (gateways []string, legacy bool) {
	raw, found := config.Lookup[any](cf, GATEWAYS, config.NoExpand())
	if !found {
		return nil, false
	}
	return gatewaysFromValue(raw)
}

func gatewaysFromValue(raw any) (gateways []string, legacy bool) {
	switch v := raw.(type) {
	case nil:
	case []string:
		gateways = slices.Clone(v)
	case Gateways:
		gateways = slices.Clone(v)
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				gateways = append(gateways, s)
			}
		}
	case string:
		for s := range strings.SplitSeq(v, ",") {
			if s = strings.TrimSpace(s); s != "" {
				gateways = append(gateways, s)
			}
		}
	case map[string]string:
		legacy = true
		for _, host := range slices.Sorted(maps.Keys(v)) {
			gateways = append(gateways, joinGatewayPort(host, v[host]))
		}
	case map[string]any:
		legacy = true
		for _, host := range slices.Sorted(maps.Keys(v)) {
			port := ""
			if v[host] != nil {
				port = fmt.Sprint(v[host])
			}
			gateways = append(gateways, joinGatewayPort(host, port))
		}
	}
	return
}

func joinGatewayPort(host, port string) string {
	if port == "" {
		return host
	}
	return host + ":" + port
}

// MigrateGateways rewrites a legacy map form of the "gateways" setting
// as a list of "host:port" strings in cf. It returns true if the
// configuration was changed.
func MigrateGateways(cf *config.Config) (changed bool) {
	gateways, legacy := GatewaysFromConfig(cf)
	if !legacy {
		return false
	}
	if gateways == nil {
		gateways = []string{}
	}
	config.Set(cf, GATEWAYS, gateways)
	return true
}

// TLSGatewayPorts applies the TLS capability of a SAN or floating probe
// to a list of "host:port" gateway connections. Only an entry using
// port 7039 is moved to 7038 when secure is true and only an entry
// using 7038 is moved to 7039 when secure is false. All other entries,
// including those with any other port or no port, are returned exactly
// as given. The result is a new slice and changed is true if any entry
// was updated.
func TLSGatewayPorts(gateways []string, secure bool) (result []string, changed bool) {
	result = make([]string, len(gateways))
	for n, entry := range gateways {
		result[n] = entry
		host, port, found := cutGatewayPort(entry)
		if !found {
			continue
		}
		switch {
		case secure && port == GatewayInsecurePort:
			result[n] = host + ":" + GatewaySecurePort
			changed = true
		case !secure && port == GatewaySecurePort:
			result[n] = host + ":" + GatewayInsecurePort
			changed = true
		}
	}
	return
}
