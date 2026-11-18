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
	"cmp"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

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

	conf, err := config.Read(cordial.ExecutableName(),
		config.UseDefaults(false),
		config.Reader(in),
		config.Format("yaml"),
	)
	if err != nil {
		log.Error("loading inventory", slog.Any("error", err))
		return nil, nil, err
	}
	contents = buf.Bytes()

	inv = &Inventory{
		inventory: conf,
		hosts:     make(map[string]string),
	}

	for _, k := range conf.AllKeys() {
		inv.hosts[k] = config.Get[string](conf, k)
	}
	return
}

// inventoryTransport returns the shared transport for remote inventories
// with TLS verification as requested. The transports are shared so that
// idle connections are reused and not leaked on each reload.
func inventoryTransport(insecure bool) *http.Transport {
	if insecure {
		return insecureInventoryTransport()
	}
	return verifiedInventoryTransport()
}

var (
	insecureInventoryTransport = sync.OnceValue(func() *http.Transport { return newInventoryTransport(true) })
	verifiedInventoryTransport = sync.OnceValue(func() *http.Transport { return newInventoryTransport(false) })
)

func newInventoryTransport(insecure bool) *http.Transport {
	t := defaultInventoryTransport().Clone()
	t.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure}
	return t
}

// ReadHostsYAML read all the inventories referenced in the YAML
// configuration and returns a consolidated map of hostname to mappings.
// If the slice "loop" exists it runs for each value, setting the
// mapping "index" to the loop value.
//
// If there are inventories configured and none of them could be read
// then an error is returned, so that callers can keep any existing
// hosts. Failures of only some inventories are logged and the hosts
// from the others are returned.
func ReadHostsYAML(cf *config.Config) (hosts map[string]HostMappings, err error) {
	hosts = make(map[string]HostMappings)

	timeout := config.Get[time.Duration](cf, "inventory.timeout")
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	httpClient := &http.Client{
		Transport: defaultInventoryTransport(),
		Timeout:   timeout,
	}
	httpsClient := &http.Client{
		Transport: inventoryTransport(config.Get[bool](cf, "inventory.insecure")),
		Timeout:   timeout,
	}

	lookup := config.Get[map[string]string](cf, "inventory.mappings")

	defFetchopts := []FetchOption{InventoryType("yaml")}

	switch config.Get[string](cf, "inventory.authentication.type") {
	case "header":
		defFetchopts = append(defFetchopts,
			AddHeader(config.Get[string](cf, "inventory.authentication.header"),
				config.Get[[]string](cf, "inventory.authentication.value"),
			))
	case "basic":
		// `user` and `pass` were used in the example configuration
		username := cmp.Or(config.Get[string](cf, "inventory.authentication.username"), config.Get[string](cf, "inventory.authentication.user"))
		password := config.Get[config.Secret](cf, "inventory.authentication.password")
		if len(password) == 0 {
			password = config.Get[config.Secret](cf, "inventory.authentication.pass")
		}
		defFetchopts = append(defFetchopts, BasicAuth(username, password))
	}

	checkModified := config.Get[bool](cf, "inventory.check-modified")
	indices := config.Get[[]string](cf, "inventory.indices")
	var loaded, failed int

	for _, index := range indices {
		var inv *Inventory
		var err error

		expand := []config.ExpandOption{
			config.LookupTable(lookup),
			config.LookupTable(map[string]string{"index": index}),
		}
		source := config.Get[string](cf, "inventory.source", expand...)

		// the same key is used to find and to store the previous state
		stateKey := source

		fetchopts := slices.Clone(defFetchopts)

		if checkModified {
			if pi, ok := Inventories.Load(stateKey); ok {
				if pinv, ok := pi.(*Inventory); ok {
					log.Debug("checking inventory", slog.String("src", source), slog.Int64("size", pinv.size), slog.Time("last_modified", pinv.lastModified))
					fetchopts = append(fetchopts, IfModified(pinv))
				}
			}
		}

		switch {
		case strings.HasPrefix(source, "http:"):
			fetchopts = append(fetchopts, Client(httpClient))
			inv, err = FetchInventory(cf, source, inventoryCache(cf, expand), fetchopts...)
		case strings.HasPrefix(source, "https:"):
			fetchopts = append(fetchopts, Client(httpsClient))
			inv, err = FetchInventory(cf, source, inventoryCache(cf, expand), fetchopts...)
		default:
			// remove file: scheme, if any
			if rest, ok := strings.CutPrefix(source, "file://"); ok {
				source = rest
			} else {
				source = strings.TrimPrefix(source, "file:")
			}
			source = config.ResolveHome(source)
			inv, err = ReadInventory(cf, source, fetchopts...)
		}

		if err != nil {
			failed++
			log.Error("failed to read inventory", slog.String("src", source), slog.Any("error", err))
			continue
		}
		loaded++

		if checkModified {
			log.Debug("storing inventory", slog.String("src", source), slog.Int64("size", inv.size), slog.Time("last_modified", inv.lastModified))
			Inventories.Store(stateKey, inv)
		}

		for h, t := range inv.hosts {
			hosts[h] = maps.Clone(lookup)
			if hosts[h] == nil {
				hosts[h] = HostMappings{}
			}
			hosts[h]["hostname"] = h
			hosts[h]["hosttype"] = t
		}
	}

	if loaded == 0 && failed > 0 {
		return nil, fmt.Errorf("none of the %d configured inventories could be read", failed)
	}

	return
}

// inventoryCache returns the expanded path to save a copy of a remote
// inventory to, if configured
func inventoryCache(cf *config.Config, expand []config.ExpandOption) string {
	cache := config.Get[string](cf, "inventory.cache", expand...)
	if cache == "" {
		return ""
	}
	return config.ResolveHome(cache)
}
