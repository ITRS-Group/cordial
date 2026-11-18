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
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
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

// maxInventorySize is the largest remote inventory that will be loaded
const maxInventorySize = 64 << 20

// maxErrorBody is the most of an error response body that is included
// in an error message
const maxErrorBody = 4 << 10

// ReadInventory reads the inventory from a file source
func ReadInventory(cf *config.Config, file string, options ...FetchOption) (inv *Inventory, err error) {
	fo := evalFetchOptions(options...)

	if fo.ifmodified != nil {
		if st, err := os.Stat(file); err == nil { // stat succeeds
			if st.Size() == fo.ifmodified.size && st.ModTime().Equal(fo.ifmodified.lastModified) {
				log.Info("inventory not modified", slog.String("file", file))
				return fo.ifmodified, nil
			}
		}
	}
	log.Info("loading inventory", slog.String("file", file))

	in, err := os.Open(file)
	if err != nil {
		log.Error("error opening inventory file", slog.Any("error", err), slog.String("file", file))
		return
	}
	defer in.Close()

	switch fo.inventoryType {
	case "yaml":
		inv, _, err = ParseInventoryYAML(cf, "", in)
	default:
		err = fmt.Errorf("unsupported inventory type %q", fo.inventoryType)
	}
	if err != nil {
		return nil, err
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
func FetchInventory(cf *config.Config, source string, cacheFile string, options ...FetchOption) (inv *Inventory, err error) {
	var cache []byte

	fo := evalFetchOptions(options...)

	req, err := http.NewRequest(fo.method, source, nil)
	if err != nil {
		return nil, err
	}
	req.Header = fo.header
	if fo.ifmodified != nil && !fo.ifmodified.lastModified.IsZero() {
		req.Header.Set("If-Modified-Since", fo.ifmodified.lastModified.Format(http.TimeFormat))
	}
	if fo.username != "" && len(fo.password) > 0 {
		req.SetBasicAuth(fo.username, string(fo.password))
	}
	req.Header.Set("Accept", "application/json")
	resp, err := fo.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		if fo.ifmodified == nil {
			return nil, fmt.Errorf("%s for an unconditional request", resp.Status)
		}
		log.Info("inventory not modified (header)", slog.String("src", source))
		inv = fo.ifmodified
		return
	}
	if fo.ifmodified != nil && fo.ifmodified.cksum != "" {
		cksum := resp.Header.Get("x-gitlab-content-sha256")
		if cksum != "" && cksum == fo.ifmodified.cksum {
			log.Info("inventory not modified (cksum)", slog.String("src", source))
			return fo.ifmodified, nil
		}
	}
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		err = fmt.Errorf("%s %s", resp.Status, string(b))
		return
	}

	log.Info("loading inventory", slog.String("src", source))

	contentLength := resp.ContentLength
	if contentLength == -1 {
		if gsize := resp.Header.Get("x-gitlab-size"); gsize != "" {
			contentLength, _ = strconv.ParseInt(gsize, 10, 64)
		}
	}

	// read a bounded amount, to avoid an unreasonable memory use from a
	// misbehaving server
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxInventorySize+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxInventorySize {
		return nil, fmt.Errorf("inventory %s is larger than %d bytes", source, maxInventorySize)
	}

	switch fo.inventoryType {
	case "yaml":
		inv, cache, err = ParseInventoryYAML(cf, cacheFile, bytes.NewReader(body))
	default:
		err = fmt.Errorf("unsupported inventory type %q", fo.inventoryType)
	}
	if err != nil {
		return nil, err
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
		if err := writeCache(cacheFile, cache); err != nil {
			log.Warn("writing inventory cache", slog.Any("error", err), slog.String("cache", cacheFile))
		}
	}

	return
}

// writeCache atomically writes data to cacheFile, creating any
// intermediate directories
func writeCache(cacheFile string, data []byte) (err error) {
	dir, file := filepath.Split(cacheFile)
	if dir == "" {
		dir = "."
	}
	if err = os.MkdirAll(dir, 0775); err != nil {
		return
	}
	f, err := os.CreateTemp(dir, file+"-*")
	if err != nil {
		return
	}
	// clean-up, which fails harmlessly once the file is renamed
	defer os.Remove(f.Name())
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return
	}
	return os.Rename(f.Name(), cacheFile)
}

// defaultInventoryTransport returns a shared transport that supports
// proxies from the environment and file: URLs. It is shared so that idle
// connections are reused and not leaked
var defaultInventoryTransport = sync.OnceValue(func() *http.Transport {
	t := &http.Transport{Proxy: http.ProxyFromEnvironment}
	t.RegisterProtocol("file", http.NewFileTransport(http.Dir("/")))
	return t
})

type fetchOptions struct {
	inventoryType string
	method        string
	client        *http.Client
	header        http.Header
	ifmodified    *Inventory
	username      string
	password      config.Secret
}

func evalFetchOptions(options ...FetchOption) (f *fetchOptions) {
	f = &fetchOptions{
		inventoryType: "yaml",
		method:        "GET",
		client:        &http.Client{},
		header:        http.Header{},
	}

	for _, opt := range options {
		opt(f)
	}

	// unless user has set the transport, use the shared one with a file
	// handler
	if f.client.Transport == nil {
		f.client.Transport = defaultInventoryTransport()
	}
	return
}

// FetchOption for FetchInventory options
type FetchOption func(*fetchOptions)

// InventoryType sets the inventory format. The default is YAML.
func InventoryType(t string) FetchOption {
	return func(fo *fetchOptions) {
		fo.inventoryType = t
	}
}

// Method sets the fetch method, defaults to GET
func Method(method string) FetchOption {
	return func(fo *fetchOptions) {
		fo.method = method
	}
}

// Client sets the http Client for FetchInventory
func Client(client *http.Client) FetchOption {
	return func(fo *fetchOptions) {
		fo.client = client
	}
}

// AddHeader adds a header to the FetchInventory call. Any existing
// header with the name name is overwritten
func AddHeader(name string, values []string) FetchOption {
	return func(fo *fetchOptions) {
		fo.header[name] = values
	}
}

// IfModified checks if the inventory inv has changed since the last
// request, using the If-Modified-Since header
func IfModified(inv *Inventory) FetchOption {
	return func(fo *fetchOptions) {
		fo.ifmodified = inv
	}
}

// BasicAuth sets up the request to use Basic Authentication
func BasicAuth(username string, password config.Secret) FetchOption {
	return func(fo *fetchOptions) {
		fo.username = username
		fo.password = password
	}
}
