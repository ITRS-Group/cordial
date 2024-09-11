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

// The inventory routines in this file handle the loading and parsing of inventory files.
//
//

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/rs/zerolog/log"
)

// Inventory type stores the results of loading an Inventory, including a host to node_type mapping
type Inventory struct {
	inventory    *config.Config    // full inventory, as loaded
	source       string            // the full name, for later lookup
	hosts        map[string]string // hostname to node_type
	size         int64
	lastModified time.Time
	cksum        string // (gitlab) cksum of file contents, for when if-modified-since r size don't work
}

// ReadInventory reads the inventory from a file source
func ReadInventory(cf *config.Config, file string, options ...FetchOptions) (inv *Inventory, err error) {
	fo := evalFetchOptions(options...)

	if fo.ifmodified != nil {
		if st, err := os.Stat(file); err == nil { // stat succeeds
			if st.Size() == fo.ifmodified.size && st.ModTime().Equal(fo.ifmodified.lastModified) {
				log.Info().Msgf("inventory not modified: %s", file)
				return fo.ifmodified, nil
			}
		}
	}
	log.Info().Msgf("loading inventory: %s", file)

	in, err := os.Open(file)
	if err != nil {
		return
	}
	defer in.Close()
	switch fo.inventoryType {
	case "yaml":
		inv, _, err = ParseInventoryYAML(cf, "", in)
	}
	inv.source = file
	if st, err := os.Stat(file); err == nil { // stat succeeds
		inv.size = st.Size()
		inv.lastModified = st.ModTime()
	}
	return
}

// FetchInventory fetches an inventory file in JSON format from the
// source URL with optional method (default GET), client and requests.
func FetchInventory(cf *config.Config, source string, cacheFile string, options ...FetchOptions) (inv *Inventory, err error) {
	var cache []byte

	fo := evalFetchOptions(options...)

	req, err := http.NewRequest(fo.method, source, nil)
	req.Header = fo.header
	if fo.ifmodified != nil && !fo.ifmodified.lastModified.IsZero() {
		req.Header["if-modified-since"] = []string{fo.ifmodified.lastModified.Format(http.TimeFormat)}
	}
	if fo.username != "" && !fo.password.IsNil() {
		req.SetBasicAuth(fo.username, fo.password.String())
	}
	req.Header["accept"] = []string{"application/json"}
	resp, err := fo.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		log.Info().Msgf("inventory not modified (header): %s", source)
		inv = fo.ifmodified
		return
	}
	if fo.ifmodified != nil && fo.ifmodified.cksum != "" {
		cksum := resp.Header.Get("x-gitlab-content-sha256")
		if cksum != "" && cksum == fo.ifmodified.cksum {
			log.Info().Msgf("inventory not modified (cksum): %s", source)
			return fo.ifmodified, nil
		}
	}
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("%s %s", resp.Status, string(b))
		return
	}

	log.Info().Msgf("loading inventory: %s", source)

	contentLength := resp.ContentLength
	if contentLength == -1 {
		if gsize := resp.Header.Get("x-gitlab-size"); gsize != "" {
			contentLength, _ = strconv.ParseInt(gsize, 10, 64)
		}
	}

	switch fo.inventoryType {
	case "yaml":
		inv, cache, err = ParseInventoryYAML(cf, cacheFile, resp.Body)
	}

	// set size and last modified if available, ignore errors as zero values are valid
	inv.source = source
	inv.size = contentLength
	inv.cksum = resp.Header.Get("x-gitlab-content-sha256")
	if lm := resp.Header.Get("last-modified"); lm != "" {
		inv.lastModified, _ = http.ParseTime(lm)
	}

	// write cache if non-zero len. any error results in clean-up and no write.
	if len(cache) > 0 && cacheFile != "" {
		dir, file := filepath.Split(cacheFile)
		if err = os.MkdirAll(dir, 0775); err != nil {
			log.Warn().Err(err).Msg("making directories")
			return inv, nil
		}
		f, err := os.CreateTemp(dir, file+"-*")
		if err != nil {
			log.Warn().Err(err).Msg("creating temp file")
			return inv, nil
		}
		// defer clean-up (which fails once file is renamed)
		defer f.Close()
		defer os.Remove(f.Name())
		if _, err = f.Write(cache); err != nil {
			log.Warn().Err(err).Msg("writing inventory to temp file")
			return inv, nil
		}
		if err = os.Rename(f.Name(), cacheFile); err != nil {
			log.Warn().Err(err).Msg("renaming temp file")
			return inv, nil
		}
	}

	return
}

type fetchOptions struct {
	inventoryType string
	method        string
	client        *http.Client
	header        http.Header
	ifmodified    *Inventory
	username      string
	password      *config.Plaintext
}

func evalFetchOptions(options ...FetchOptions) (f *fetchOptions) {
	f = &fetchOptions{
		inventoryType: "yaml",
		method:        "GET",
		client:        &http.Client{},
		header:        http.Header{},
	}

	for _, opt := range options {
		opt(f)
	}

	// unless user has set the transport, add a file handler
	if f.client.Transport == nil {
		t := &http.Transport{}
		t.RegisterProtocol("file", http.NewFileTransport(http.Dir("/")))
		f.client.Transport = t
	}
	return
}

// FetchOptions for FetchInventory options
type FetchOptions func(*fetchOptions)

// InventoryType sets the inventory format. The default is YAML.
func InventoryType(t string) FetchOptions {
	return func(fo *fetchOptions) {
		fo.inventoryType = t
	}
}

// Method sets the fetch method, defaults to GET
func Method(method string) FetchOptions {
	return func(fo *fetchOptions) {
		fo.method = method
	}
}

// Client sets the http Client for FetchInventory
func Client(client *http.Client) FetchOptions {
	return func(fo *fetchOptions) {
		fo.client = client
	}
}

// AddHeader adds a header to the FetchInventory call. Any existing
// header with the name name is overwritten
func AddHeader(name string, values []string) FetchOptions {
	return func(fo *fetchOptions) {
		fo.header[name] = values
	}
}

// IfModified checks if the inventory inv has changed since the last
// request, using the If-Modified-Since header
func IfModified(inv *Inventory) FetchOptions {
	return func(fo *fetchOptions) {
		fo.ifmodified = inv
	}
}

// BasicAuth sets up the request to use Basic Authentication
func BasicAuth(username string, password *config.Plaintext) FetchOptions {
	return func(fo *fetchOptions) {
		fo.username = username
		fo.password = password
	}
}
