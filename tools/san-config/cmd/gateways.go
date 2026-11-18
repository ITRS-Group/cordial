/*
Copyright © 2024 ITRS Group

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

package cmd

import (
	"cmp"
	"context"
	"crypto/tls"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/itrs-group/cordial/pkg/config"
)

// GatewaySet is a Gateway instance with standby details if required
type GatewaySet struct {
	Name        string
	Primary     string
	PrimaryPort int
	Standby     string
	StandbyPort int
	Secure      bool
}

type gatewayList struct {
	sync.Mutex
	gateways []string
}

// the clients for the liveness checks are shared between calls so that
// idle connections are reused and not leaked. The timeout is applied to
// each request instead of the client, as it can be changed in the
// configuration.
var (
	livenessHTTPClient  = &http.Client{}
	livenessHTTPSClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // we just want to know if it responds, let the client check certs
			},
		},
	}
)

// withDefaultPort returns hostport with the default Gateway port, which
// depends on whether the connection is secure, added if it has none
func withDefaultPort(hostport string, secure bool) string {
	if strings.Contains(hostport, ":") {
		return hostport
	}
	if secure {
		return hostport + ":7038"
	}
	return hostport + ":7039"
}

// CheckGateways uses a liveness endpoint to see if gateway is
// reachable. primary and standby gateways should normally respond the
// same, and as long as one of each pair is "up" we respond OK for the
// set.
//
// all gateways are checked in their own go routines
func CheckGateways(cf *config.Config) (liveGateways []string) {
	var gateways gatewayList
	var wg sync.WaitGroup
	var totalCount int

	timeout := config.Get[time.Duration](cf, "geneos.timeout")

	for i, g := range config.Get[[]map[string]string](cf, "geneos.gateways") {
		if g["name"] == "" && g["primary"] == "" {
			log.Debug("no name or primary defined for gateway, skipping", slog.Int("index", i))
			continue
		}

		primary := g["primary"]
		if primary == "" {
			// "name" must be defined else we would have skipped above
			primary = g["name"]
		}
		totalCount++

		standby := g["standby"]
		name, ok := g["name"]
		if !ok {
			name = primary
		}

		secure, err := strconv.ParseBool(g["secure"])
		if err != nil {
			log.Debug("gateway secure setting unknown, assuming false", slog.String("name", name))
			secure = false
		}

		log.Debug("gateway details", slog.String("primary", primary), slog.String("standby", standby), slog.Bool("secure", secure))

		client := livenessHTTPSClient
		scheme := "https"
		if !secure {
			client = livenessHTTPClient
			scheme = "http"
		}

		primaryURL := url.URL{
			Scheme: scheme,
			Host:   withDefaultPort(primary, secure),
			Path:   "/liveness",
		}
		wg.Go(func() { checkGateway(client, timeout, &gateways, name, "primary", primaryURL) })

		if standby == "" {
			continue
		}

		standbyURL := url.URL{
			Scheme: scheme,
			Host:   withDefaultPort(standby, secure),
			Path:   "/liveness",
		}
		wg.Go(func() { checkGateway(client, timeout, &gateways, name, "standby", standbyURL) })
	}

	wg.Wait()
	liveGateways = gateways.gateways

	if len(liveGateways) == totalCount {
		log.Info("gateway sets are available", slog.Int("available", len(liveGateways)), slog.Int("total", totalCount))
	} else {
		log.Warn("gateway sets are not fully available", slog.Int("available", len(liveGateways)), slog.Int("total", totalCount))

	}

	return
}

func checkGateway(client *http.Client, timeout time.Duration, gateways *gatewayList, name, role string, livenessURL url.URL) {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, livenessURL.String(), nil)
	if err != nil {
		log.Warn("gateway check failed", slog.String("name", name), slog.String("role", role), slog.String("host", livenessURL.Host), slog.Any("error", err))
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Warn("gateway not responding", slog.String("name", name), slog.String("role", role), slog.String("host", livenessURL.Host))
		return
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Warn("gateway returned unexpected status", slog.String("name", name), slog.String("role", role), slog.String("host", livenessURL.Host), slog.Int("status", resp.StatusCode), slog.String("status_text", resp.Status))
		return
	}
	// add to list
	log.Debug("gateway responding to liveness check", slog.String("name", name), slog.String("role", role), slog.String("host", livenessURL.Host))
	gateways.Lock()
	if !slices.Contains(gateways.gateways, name) {
		gateways.gateways = append(gateways.gateways, name)
	}
	gateways.Unlock()
}

// OrderGateways creates an ordered list of the hash of probe and
// gateway names. The hash is created through UUIDs based on the
// configuration namespace uuid.
func OrderGateways(netprobe string, gateways []string) (selection []string) {
	if len(gateways) == 0 {
		return
	}

	probeUUID := uuid.NewSHA1(uuidNS, []byte(netprobe))

	gws := map[string]string{}
	gwUUIDs := []string{}

	for _, g := range gateways {
		gwUUID := uuid.NewSHA1(probeUUID, []byte(g)).String()
		gws[gwUUID] = g
		gwUUIDs = append(gwUUIDs, gwUUID)
	}

	log.Debug("uuids", slog.Any("uuids", gwUUIDs))

	for _, k := range slices.Sorted(maps.Keys(gws)) {
		selection = append(selection, gws[k])
	}

	return
}

// GatewayDetails searches for and returns the details for Gateway name.
// As for the liveness checks, if there is no primary then the name is
// used as the primary host.
func GatewayDetails(name string, gateways []map[string]string) (gateway GatewaySet) {
	if name == "" {
		return
	}

	for _, g := range gateways {
		primary := cmp.Or(g["primary"], g["name"])
		if name == g["name"] || name == primary {
			gateway.Name = name
			gateway.Secure, _ = strconv.ParseBool(g["secure"])

			host, port, found := strings.Cut(primary, ":")
			gateway.Primary = host
			if found {
				gateway.PrimaryPort, _ = strconv.Atoi(port)
			}
			if gateway.PrimaryPort == 0 {
				if gateway.Secure {
					gateway.PrimaryPort = 7038
				} else {
					gateway.PrimaryPort = 7039
				}
			}

			host, port, found = strings.Cut(g["standby"], ":")
			gateway.Standby = host
			if found {
				gateway.StandbyPort, _ = strconv.Atoi(port)
			}
			if gateway.StandbyPort == 0 {
				if gateway.Secure {
					gateway.StandbyPort = 7038
				} else {
					gateway.StandbyPort = 7039
				}
			}

			return
		}
	}
	return
}
