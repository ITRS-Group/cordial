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
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"sync/atomic"

	"github.com/rs/zerolog/log"
)

// counter is used to generate unique temporary file names when saving
// configuration files. It is an atomic counter to ensure that it can be
// safely incremented across multiple goroutines without the need for
// additional synchronization mechanisms. The counter is used in
// conjunction with the process ID to create a unique temporary file
// name for each save operation, which helps to avoid conflicts and
// ensure that concurrent save operations do not interfere with each
// other.
var counter atomic.Int64

// Save a configuration file for the named module.
//
// - The file specified by config.SetConfigFile() - A file name.ext in
// the first directory give with config.AddDirs() - A file name.ext in
// the user config directory + appname
//
// The filesystem target for the configuration object is updated to
// match the remote destination, which can be set by Host() option with
// a default of "localhost"
//
// Save writes to a temporary file with the process ID and original
// extension appended and then tries to atomically rename the file to
// the target name. This is to avoid leaving a partially written file if
// the write operation is interrupted. It may result in a temporary file
// being left behind if the rename operation fails, but this is
// preferable to leaving a corrupted configuration file. The temporary
// file is named with the process ID to avoid conflicts with other
// processes that may be writing to the same configuration file. The
// original extension is preserved to ensure that the temporary file is
// recognized as a configuration file by any tools that may be
// monitoring the directory for changes.
func (c *Config) Save(name string, options ...FileOptions) (err error) {
	log.Debug().Msg("saving configuration")
	var p string

	opts := evalSaveOptions(name, options...)
	h := opts.remote

	if ok, err := h.IsAvailable(); !ok {
		return err
	}

	filename := fmt.Sprintf("%s.%s", name, opts.extension)

	if opts.userConfDir != "" {
		p = path.Join(opts.userConfDir, opts.appName, filename)
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

	if err = h.MkdirAll(path.Dir(p), 0775); err != nil {
		return
	}

	// copy all keys and values to a new config object to avoid
	// modifying the original config with any expansions or
	// transformations needed for saving. This is to ensure that the
	// original config remains unchanged and can be used for other
	// purposes without any unintended side effects.

	nv := New()

	c.mutex.RLock()
	for _, k := range c.allKeys() {
		if slices.Contains(opts.ignoreKeys, k) {
			continue
		}
		v := c.get(k)
		// if given the IgnoreEmptyValues option, skip aliases and keys
		// with zero/empty values
		if opts.ignoreEmptyValues && isZero(v) {
			continue
		}
		nv.Set(k, v)
	}
	c.mutex.RUnlock()

	nv.Viper.SetFs(h.GetFs())

	// set the config type as well as the extension to ensure the correct format is used when writing the file
	nv.Viper.SetConfigFile(filepath.Ext(p))

	// write to path with process ID and original extension appended and
	// then try to atomically rename
	tmpFile := fmt.Sprintf("%s.%d.%d%s", p, os.Getpid(), counter.Add(1), filepath.Ext(p))
	if err = nv.Viper.WriteConfigAs(tmpFile); err != nil {
		return
	}

	// if write is OK, rename the temp file
	if err = h.Rename(tmpFile, p); err != nil { // try to remove tmp file is rename fails
		err = h.Remove(tmpFile)
	}

	return
}

// Save writes the global configuration to a configuration file defined
// by the component name and options
func Save(name string, options ...FileOptions) (err error) {
	return global.Save(name, options...)
}

// SaveTo writes the configuration to the provided writer. The name and
// options are used to determine the format of the configuration file,
// but the actual output is written to the provided writer instead of a
// file. This can be used to write the configuration to a different
// destination, such as a network connection or an in-memory buffer,
// rather than a file on disk.
//
// options can include both FileOptions and ExpandOptions, which are
// used to determine the format of the configuration file and to expand
// any dynamic values in the configuration, respectively. The function
// will handle both types of options and apply them as needed when
// writing the configuration to the provided writer.
func (c *Config) SaveTo(name string, w io.Writer, options ...FileOptions) (err error) {
	log.Debug().Msg("saving configuration")

	opts := evalSaveOptions(name, options...)

	nv := New()

	c.mutex.RLock()
	for _, k := range c.allKeys() {
		if slices.Contains(opts.ignoreKeys, k) {
			continue
		}
		v := c.get(k)
		if opts.ignoreEmptyValues && isZero(v) {
			continue
		}
		if opts.expandOnSave {
			log.Debug().Msgf("expanding key: %s", k)
			// test setting numbers
			s := c.expandString(c.GetString(k, opts.expandOptions...))
			nv.Set(k, s)
		} else {
			nv.Set(k, v)
		}
	}
	c.mutex.RUnlock()

	if opts.userConfDir != "" {
		nv.Viper.SetConfigType(opts.extension)
	} else {
		nv.Viper.SetConfigType(defaultFileExtension)
	}

	return nv.Viper.WriteConfigTo(w)
}

// isZero checks if the given value is the zero value for its type. If
// the value is not valid return false, else return the result if
// reflect.Value.IsZero()
func isZero(n any) bool {
	v := reflect.ValueOf(n)
	if v.IsValid() && v.IsZero() {
		return true
	}
	return false
}
