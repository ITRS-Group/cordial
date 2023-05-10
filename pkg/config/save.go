/*
Copyright © 2023 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package config

import (
	"fmt"
	"path/filepath"

	"github.com/itrs-group/cordial/pkg/host"
)

// Save a configuration file for the component name. The filesystem
// target for the configuration object is updated to match the remote
// destination, which can be set by SaveTo() or defaults to "localhost"
func (cf *Config) Save(name string, options ...SaveOptions) (err error) {
	opts := evalSaveOptions(options...)
	r := opts.remote
	if !r.IsAvailable() {
		err = host.ErrNotAvailable
		return
	}

	subdir := name
	if opts.appname != "" {
		subdir = opts.appname
	}
	path := filepath.Join(opts.dir, subdir, fmt.Sprintf("%s.%s", name, opts.format))

	if opts.path != "" {
		path = opts.path
	}

	if err = r.MkdirAll(filepath.Dir(path), 0775); err != nil {
		return
	}

	cf.SetFs(r.GetFs())
	return cf.WriteConfigAs(path)
}

// Save writes the global configuration to a configuration file defined
// by the component name and options
func Save(name string, options ...SaveOptions) (err error) {
	return global.Save(name, options...)
}

type saveOptions struct {
	path    string
	dir     string
	appname string
	format  string
	remote  host.Host
}

type SaveOptions func(*saveOptions)

func evalSaveOptions(options ...SaveOptions) (c *saveOptions) {
	c = &saveOptions{
		format: "json",
		remote: host.Localhost,
	}
	c.dir, _ = UserConfigDir()

	for _, opt := range options {
		opt(c)
	}

	return
}

func SaveAppName(name string) SaveOptions {
	return func(so *saveOptions) {
		so.appname = name
	}
}

func SaveInDir(dir string) SaveOptions {
	return func(so *saveOptions) {
		so.dir = dir
	}
}

func SaveInFile(path string) SaveOptions {
	return func(so *saveOptions) {
		so.path = path
	}
}

func SaveFormat(format string) SaveOptions {
	return func(so *saveOptions) {
		so.format = format
	}
}

func SaveTo(r host.Host) SaveOptions {
	return func(so *saveOptions) {
		so.remote = r
	}
}
