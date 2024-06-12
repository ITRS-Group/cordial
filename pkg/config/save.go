/*
Copyright © 2023 ITRS Group

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

package config

import (
	"fmt"
	"os"
	"path"
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
	var ok bool
	if ok, err = r.IsAvailable(); !ok {
		return
	}

	filename := fmt.Sprintf("%s.%s", name, opts.extension)

	var p string
	if opts.userconfdir != "" {
		p = path.Join(opts.userconfdir, opts.appname, filename)
	}

	if len(opts.configDirs) > 0 {
		p = path.Join(opts.configDirs[0], filename)
	}

	if opts.configFile != "" {
		p = opts.configFile
	}

	if p == "" {
		return fmt.Errorf("cannot resolve save location: %w", os.ErrNotExist)
	}

	if err = r.MkdirAll(path.Dir(p), 0775); err != nil {
		return
	}

	cf.mutex.Lock()
	cf.Viper.SetFs(r.GetFs())
	err = cf.Viper.WriteConfigAs(p)
	cf.mutex.Unlock()
	return
}

// Save writes the global configuration to a configuration file defined
// by the component name and options
func Save(name string, options ...FileOptions) (err error) {
	return global.Save(name, options...)
}
