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
	"encoding/csv"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

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

	for i, g := range cf.GetSliceStringMapString("geneos.gateways") {
		if g["name"] == "" && g["primary"] == "" {
			log.Debug().Msgf("no name or primary defined for gateway %d, skipping", i)
		}

		primary := g["primary"]
		if primary == "" {
			// "name" must be defined else we would have failed to check above
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
			log.Debug().Msgf("gateway %q secure setting unknown, assuming false", name)
			secure = false
		}

		log.Debug().Msgf("primary %s, standby %s, secure %v", primary, standby, secure)

		if !strings.Contains(primary, ":") {
			// append default port depending on secure flag
			if secure {
				primary += ":7038"
			} else {
				primary += ":7039"
			}
		}

		primaryURL := url.URL{
			Scheme: "https",
			Host:   primary,
			Path:   "/liveness",
		}
		client := httpsClient

		if !secure {
			primaryURL.Scheme = "http"
			client = httpClient
		}

		wg.Add(1)
		go checkGateway(client, &wg, &gateways, name, "primary", primaryURL)

		if standby == "" {
			continue
		}

		if !strings.Contains(standby, ":") {
			// append default port depending on secure flag
			if secure {
				standby += ":7038"
			} else {
				standby += ":7039"
			}
		}
		standbyURL := url.URL{
			Scheme: "https",
			Host:   standby,
			Path:   "/liveness",
		}
		client = httpsClient

		if !secure {
			standbyURL.Scheme = "http"
			client = httpClient
		}

		wg.Add(1)
		go checkGateway(client, &wg, &gateways, name, "standby", standbyURL)
	}

	wg.Wait()
	liveGateways = gateways.gateways

	if len(liveGateways) == totalCount {
		log.Info().Msgf("%d/%d gateway sets are available", len(liveGateways), totalCount)
	} else {
		log.Warn().Msgf("%d/%d gateway sets are available", len(liveGateways), totalCount)

	}

	return
}

func checkGateway(client http.Client, wg *sync.WaitGroup, gateways *gatewayList, name, role string, livenessURL url.URL) {
	defer wg.Done()

	resp, err := client.Get(livenessURL.String())
	if err != nil {
		log.Warn().Msgf("gateway %s %s %s not responding", name, role, livenessURL.Host)
		return
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Warn().Msgf("gateway %s %s %s returned: %d %s", name, role, livenessURL.Host, resp.StatusCode, resp.Status)
		return
	}
	// add to list
	log.Debug().Msgf("gateway %s %s %s responding to liveness check", name, role, livenessURL.Host)
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

	log.Debug().Msgf("uuids: %v", gwUUIDs)

	for _, k := range slices.Sorted(maps.Keys(gws)) {
		selection = append(selection, gws[k])
	}

	return
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

			return
		}
	}
	return
}

type gatewayEntry struct {
	Name    string `json:"name" yaml:"name"`
	Primary string `json:"primary" yaml:"primary"`
	Standby string `json:"standby" yaml:"standby"`
	Secure  bool   `json:"secure" yaml:"secure"`
}

// ReadGateways reads an local file for gateway details. Paths can use
// the "~/" prefix for the user's home directory.
//
// CSV, JSON and YAML are supported. The file type is wholly determined
// by the source file extension, which defaults to YAML if not given, so
// ".yml" will work too.
//
// CSV files must have a first line with column names, and the column
// "name" is required while "primary", "standby" and "secure" are
// optional. The format of each field is as for the main configuration
// file.
//
// JSON files must be an array of objects, where each object has a
// required "name" field and optional fields as for CSV.
//
// YAML files must have a top-level "gateways" parameter and an array of
// object as per the main configuration file.
func ReadGateways(source string) (gateways []map[string]string) {
	source = config.ExpandHome(source)
	// try to open file
	r, err := os.Open(source)
	if err != nil {
		log.Error().Err(err).Msgf("opening gateways file %q", source)
		return
	}
	defer r.Close()

	switch filepath.Ext(source) {
	case ".csv":
		c := csv.NewReader(r)
		columns, err := c.Read()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Error().Err(err).Msg("")
			}
			return
		}
		colNames := make(map[int]string)
		for i, cn := range columns {
			colNames[i] = cn
		}
		for {
			row, err := c.Read()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					log.Error().Err(err).Msg("")
				}
				return
			}
			gw := make(map[string]string)
			for i, f := range row {
				switch colNames[i] {
				case "name", "primary", "standby", "secure":
					gw[colNames[i]] = f
				default:
					// do nothing
				}

			}
			if gw["name"] == "" {
				line, _ := c.FieldPos(1)
				log.Error().Msgf("no gateway name in %s on line %d", source, line)
			}
			gateways = append(gateways, gw)
		}
	case ".json":

	default:
		// everything else is treated as YAML
	}
	return
}
