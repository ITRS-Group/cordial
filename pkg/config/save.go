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
	"path/filepath"
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

// Write a configuration file for the named module.
//
// The configuration can be written to an io.Writer, using
// config.Writer() option, or to a file determined by the following
// order of precedence:
//
//   - The file specified by config.SetConfigFile()
//   - A file name.ext in the first directory give with config.AddDirs()
//   - A file name.ext in the user config directory + appname
//
// The filesystem target for the configuration object is updated to
// match the remote destination, which can be set by Host() option with
// a default of "localhost"
//
// Write writes to a temporary file with the process ID and original
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
func (c *Config) Write(module string, options ...FileOption) (err error) {
	var p string

	log.Debug().Msg("saving configuration")

	opts := evalSaveOptions(module, options...)

	h := opts.remote

	w := opts.writer

	if w == nil {
		if ok, err := h.IsAvailable(); !ok {
			return err
		}

		filename := fmt.Sprintf("%s.%s", module, opts.format)

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
	}

	// copy all keys and values to a new config object to avoid
	// modifying the original config with any expansions or
	// transformations needed for saving.

	nv := New()

	c.rwmutex.RLock()
	for _, k := range c.allKeys() {
		if slices.Contains(opts.ignoreKeys, k) {
			continue
		}
		v := get[any](c, k)

		// if given the IgnoreEmptyValues option, skip aliases and keys
		// with zero/empty values
		if opts.ignoreEmptyValues && isEmpty(v) {
			continue
		}
		if opts.expandOnSave {
			// test setting numbers
			set(nv, k, get[string](c, k, opts.expandOptions...))
		} else {
			set(nv, k, v)
		}
	}
	c.rwmutex.RUnlock()

	nv.setConfigType(opts.format)

	if w != nil {
		return nv.writeConfigTo(w)
	}

	nv.setFs(h.GetFs())

	// write to path with process ID and original extension appended and
	// then try to atomically rename
	tmpFile := fmt.Sprintf("%s.%d.%d%s", p, os.Getpid(), counter.Add(1), filepath.Ext(p))
	if err = nv.writeConfigAs(tmpFile); err != nil {
		return
	}

	// if write is OK, rename the temp file
	if err = h.Rename(tmpFile, p); err != nil { // try to remove tmp file is rename fails
		err = h.Remove(tmpFile)
	}

	return
}
