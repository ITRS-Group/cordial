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
	"bytes"
	"crypto/tls"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
)

const bufSize = 256 * 1024

// ParseInventoryYAML loads a simple, flat yaml style file of:
//
// name: type
// name: type
func ParseInventoryYAML(cf *config.Config, cacheFile string, in io.Reader) (inv *Inventory, contents []byte, err error) {
	buf := &bytes.Buffer{}

	if cacheFile != "" {
		buf = bytes.NewBuffer(make([]byte, 0, bufSize))
		in = io.TeeReader(in, buf)
	}

	conf, err := config.Load(cordial.ExecutableName(),
		config.UseDefaults(false),
		config.SetConfigReader(in),
		config.SetFileExtension("yaml"),
	)
	if err != nil {
		log.Error().Err(err).Msg("loading inventory")
		return
	}
	contents = buf.Bytes()

	inv = &Inventory{
		inventory: conf,
		hosts:     make(map[string]string),
	}

	for _, k := range conf.AllKeys() {
		inv.hosts[k] = conf.GetString(k)
	}
	return
}

// ReadHostsYAML read all the inventories referenced in the YAML
// configuration and returns a consolidated map of hostname to mappings.
// If the slice "loop" exists it runs for each value, setting the
// mapping "index" to the loop value.
func ReadHostsYAML(cf *config.Config) (hosts map[string]HostMappings, err error) {
	hosts = make(map[string]HostMappings)

	timeout := cf.GetDuration("inventory.timeout")
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	httpClient := &http.Client{
		Timeout: timeout,
	}
	httpsClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cf.GetBool("inventory.insecure"),
			},
		},
		Timeout: timeout,
	}

	lookup := cf.GetStringMapString("inventory.mappings")

	defFetchopts := []FetchOptions{InventoryType("yaml")}

	switch cf.GetString("inventory.authentication.type") {
	case "header":
		defFetchopts = append(defFetchopts,
			AddHeader(cf.GetString("inventory.authentication.header"),
				cf.GetStringSlice("inventory.authentication.value"),
			))
	case "basic":
		defFetchopts = append(defFetchopts,
			BasicAuth(cf.GetString("inventory.authentication.username"),
				cf.GetPassword("inventory.authentication.password"),
			))
	}

	for _, index := range cf.GetStringSlice("inventory.indices") {
		var inv *Inventory
		var err error

		source := cf.GetString("inventory.source",
			config.LookupTable(lookup),
			config.LookupTable(map[string]string{"index": index}),
		)

		fetchopts := slices.Clone(defFetchopts)

		if cf.GetBool("inventory.check-modified") {
			if pi, ok := Inventories.Load(source); ok {
				if pinv, ok := pi.(*Inventory); ok {
					log.Debug().Msgf("checking inventory %s - size %d, last mod %s", source, pinv.size, pinv.lastModified.Format(time.RFC3339))
					fetchopts = append(fetchopts, IfModified(pinv))
				}
			}
		}

		switch {
		case strings.HasPrefix(source, "http:"):
			fetchopts = append(fetchopts, Client(httpClient))
			inv, err = FetchInventory(cf, source, "", fetchopts...)
		case strings.HasPrefix(source, "https:"):
			fetchopts = append(fetchopts, Client(httpsClient))
			inv, err = FetchInventory(cf, source, "", fetchopts...)
		case strings.HasPrefix(source, "file"):
			// remove file: scheme and drop through
			source = strings.TrimPrefix(source, "file:")
			fallthrough
		default:
			source = config.ExpandHome(source)
			inv, err = ReadInventory(cf, source)
		}

		if err != nil {
			log.Error().Err(err).Msgf("failed to read inventory from %s", source)
			continue
		}

		if cf.GetBool("inventory.check-modified") {
			log.Debug().Msgf("storing inventory for %s - size %d, last mod %s", source, inv.size, inv.lastModified.Format(time.RFC3339))
			Inventories.Store(source, inv)
		}

		for h, t := range inv.hosts {
			hosts[h] = maps.Clone(lookup)
			hosts[h]["hostname"] = h
			hosts[h]["hosttype"] = t
		}
	}

	return
}
