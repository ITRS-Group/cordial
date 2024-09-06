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
	"crypto/tls"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
)

// CheckGateways uses a liveness endpoint to see if gateway is
// reachable. primary and standby gateways should normally respond the
// same, and as long as one of each pair is "up" we respond OK for the
// set.
//
// all gateways are checked in their own go routines
func CheckGateways(cf *config.Config) (liveGateways []string) {
	var mutex sync.Mutex
	var wg sync.WaitGroup
	var totalCount int

	httpClient := http.Client{
		Timeout: cf.GetDuration("geneos.timeout"),
	}
	httpsClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // we just want to know if it responds, let the client check certs
			},
		},
		Timeout: cf.GetDuration("geneos.timeout"),
	}

	for _, g := range cf.GetSliceStringMapString("geneos.gateways") {
		primary := g["primary"]
		if primary == "" {
			continue
		}
		totalCount++
		standby := g["standby"]
		secure := g["secure"]
		name, ok := g["name"]
		if !ok {
			name = primary
		} else {
			totalCount++
		}

		log.Debug().Msgf("primary %s, standby %s, secure %s", primary, standby, secure)

		primaryURL := url.URL{
			Scheme: "https",
			Host:   primary,
			Path:   "/liveness",
		}
		client := httpsClient

		if secure != "1" {
			primaryURL.Scheme = "http"
			client = httpClient
		}

		wg.Add(1)
		go checkGateway(client, &wg, &mutex, &liveGateways, name, primaryURL)

		if standby == "" {
			continue
		}

		standbyURL := url.URL{
			Scheme: "https",
			Host:   standby,
			Path:   "/liveness",
		}
		client = httpsClient

		if secure != "1" {
			standbyURL.Scheme = "http"
			client = httpClient
		}

		wg.Add(1)
		go checkGateway(client, &wg, &mutex, &liveGateways, name, standbyURL)
	}

	wg.Wait()
	if len(liveGateways) == totalCount {
		log.Info().Msgf("%d/%d gateways are available", len(liveGateways), totalCount)
	} else {
		log.Warn().Msgf("%d/%d gateways are available", len(liveGateways), totalCount)

	}

	return
}

func checkGateway(client http.Client, wg *sync.WaitGroup, mutex *sync.Mutex, liveGateways *[]string, name string, livenessURL url.URL) {
	defer wg.Done()

	resp, err := client.Get(livenessURL.String())
	if err != nil {
		log.Warn().Msgf("gateway %s %s not responding", name, livenessURL.Host)
		return
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Warn().Msgf("gateway %s %s returned: %d %s", name, livenessURL.Host, resp.StatusCode, resp.Status)
		return
	}
	// add to list
	log.Debug().Msgf("gateway %s %s responding to liveness check", name, livenessURL.Host)
	mutex.Lock()
	*liveGateways = append(*liveGateways, name)
	mutex.Unlock()
}

// SelectGateway creates an ordered list of the hash of probe and
// gateway names and returns the first one. The hash is created through
// UUIDs based on the configuration namespace uuid.
func SelectGateway(probename string, gateways []string) (gateway string) {
	if len(gateways) == 0 {
		return
	}

	probeUUID := uuid.NewSHA1(uuidNS, []byte(probename))

	gws := map[string]string{}
	gwUUIDs := []string{}

	for _, g := range gateways {
		gwUUID := uuid.NewSHA1(probeUUID, []byte(g)).String()
		gws[gwUUID] = g
		gwUUIDs = append(gwUUIDs, gwUUID)
	}

	log.Debug().Msgf("uuids: %v", gwUUIDs)
	first := slices.Min(gwUUIDs)

	log.Debug().Msgf("selecting %s -> %s", first, gws[first])
	return gws[first]
}

// GatewaySet is a Gateway instance with standby details if required
type GatewaySet struct {
	Name        string
	Primary     string
	PrimaryPort int
	Standby     string
	StandbyPort int
	Secure      bool
}

// GatewayDetails searches for and returns the details for Gateway name
func GatewayDetails(name string, gateways []map[string]string) (gateway GatewaySet) {
	for _, g := range gateways {
		if name == g["name"] || name == g["primary"] {
			gateway.Name = name
			gateway.Secure, _ = strconv.ParseBool(g["secure"])

			p := strings.SplitN(g["primary"], ":", 2)
			gateway.Primary = p[0]
			if len(p) > 1 {
				gateway.PrimaryPort, _ = strconv.Atoi(p[1])
			}
			if gateway.PrimaryPort == 0 {
				if gateway.Secure {
					gateway.PrimaryPort = 7038
				} else {
					gateway.PrimaryPort = 7039
				}
			}

			s := strings.SplitN(g["standby"], ":", 2)
			gateway.Standby = s[0]
			if len(s) > 1 {
				gateway.StandbyPort, _ = strconv.Atoi(s[1])
			}
			if gateway.StandbyPort == 0 {
				if gateway.Secure {
					gateway.StandbyPort = 7038
				} else {
					gateway.StandbyPort = 7039
				}
			}
			break
		}
	}
	return
}
