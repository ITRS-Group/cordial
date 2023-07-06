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
	"path"
	"path/filepath"

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/rs/zerolog/log"
)

// Save a configuration file for the module name.
//
// - The file specified by config.SetConfigFile()
// - A file name.ext in the first directory give with config.AddDirs()
// - A file name.ext in the user config directory + appname
//
// The filesystem target for the configuration object is updated to
// match the remote destination, which can be set by Host() option with
// a default of "localhost"
func (cf *Config) Save(name string, options ...FileOptions) (err error) {
	opts := evalSaveOptions(name, options...)
	r := opts.remote
	if !r.IsAvailable() {
		err = host.ErrNotAvailable
		return
	}

	filename := fmt.Sprintf("%s.%s", name, opts.extension)

	p := path.Join(opts.userconfdir, opts.appname, filename)
	if len(opts.configDirs) > 0 {
		p = path.Join(opts.configDirs[0], filename)
	}

	if opts.configFile != "" {
		p = opts.configFile
	}

	if err = r.MkdirAll(filepath.Dir(p), 0775); err != nil {
		return
	}

	cf.SetFs(r.GetFs())
	log.Debug().Msgf("saving configuration to %s", p)
	return cf.WriteConfigAs(p)
}

// Save writes the global configuration to a configuration file defined
// by the component name and options
func Save(name string, options ...FileOptions) (err error) {
	return global.Save(name, options...)
}
