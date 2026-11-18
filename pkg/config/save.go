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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
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
// All of the settings of c are written, including any defaults and
// values from the environment that are visible through c, unless
// excluded with [IgnoreKeys] or [OmitEmptyValues]. Options such as
// [WithDefaults] and [WithEnvs] are not applied to what is written.
//
// New files are created with permissions 0644 and any new directories
// with 0775, before the process umask is applied. An existing file
// keeps its permissions. Use [FileMode] and [DirMode] to override these,
// e.g. for files containing credentials.
//
// Write writes to a temporary file with the process ID and original
// extension appended and then tries to atomically rename the file to
// the target name. This is to avoid leaving a partially written file if
// the write operation is interrupted. The temporary file is removed if
// the write or the rename fails. The temporary file is named with the
// process ID to avoid conflicts with other processes that may be
// writing to the same configuration file. The format of the file
// written is the extension of the target file, if it is a supported
// one, otherwise the one set by [Format], and the temporary file
// always ends in the same extension to ensure that it is
// recognised as a configuration file by any tools that may be
// monitoring the directory for changes.
func (c *Config) Write(module string, options ...FileOption) (err error) {
	var p string

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

		dirMode := opts.dirMode
		if dirMode == 0 {
			dirMode = 0775
		}
		if err = h.MkdirAll(path.Dir(p), dirMode); err != nil {
			return
		}
	}

	// copy all keys and values to a new config object to avoid
	// modifying the original config with any expansions or
	// transformations needed for saving. the keys come from c so the
	// delimiter must match.
	nv := newScratch(append(slices.Clone(options), KeyDelimiter(c.delimiter))...)

	c.rwmutex.RLock()
	expandOpts := evalExpandOptions(c, opts.expandOnSaveOptions...)
	for _, k := range c.allKeys() {
		if ignoredKey(k, c.delimiter, opts.ignoreKeys) {
			continue
		}
		v := get[any](c, k, NoExpand())

		// if given the IgnoreEmptyValues option, skip aliases and keys
		// with zero/empty values
		if opts.ignoreEmptyValues && isEmpty(v) {
			continue
		}
		if opts.expandOnSave {
			// expand strings, but keep the types of all other values
			v = expandValue(c, expandOpts, v)
		}
		set(nv, k, v)
	}
	c.rwmutex.RUnlock()

	nv.setConfigType(opts.format)

	if w != nil {
		return nv.writeConfigTo(w)
	}

	nv.setFs(h.GetFs())
	mode := opts.fileMode
	if mode == 0 {
		// keep the permissions of an existing file
		if info, statErr := h.Stat(p); statErr == nil {
			mode = info.Mode().Perm()
		}
	}
	if mode != 0 {
		nv.config.SetConfigPermissions(mode)
	}

	// the format of the file is taken from the extension of the file
	// name, so make sure the temporary file has a usable one even if the
	// target does not
	ext := strings.TrimPrefix(filepath.Ext(p), ".")
	if !slices.Contains(SupportedExts, ext) {
		ext = opts.format
	}

	// write to path with process ID and original extension appended and
	// then try to atomically rename
	tmpFile := fmt.Sprintf("%s.%d.%d.%s", p, os.Getpid(), counter.Add(1), ext)
	if err = nv.writeConfigAs(tmpFile); err != nil {
		if rmErr := h.Remove(tmpFile); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
			err = errors.Join(err, rmErr)
		}
		return
	}

	// if write is OK, rename the temp file
	if err = h.Rename(tmpFile, p); err != nil { // try to remove tmp file is rename fails
		err = errors.Join(err, h.Remove(tmpFile))
	}

	return
}

// ignoredKey returns true if the key k matches, case insensitively, any
// of ignore or is below any of them, i.e. `a.b.c` is ignored by `a.b`
func ignoredKey(k, delimiter string, ignore []string) bool {
	k = strings.ToLower(k)
	return slices.ContainsFunc(ignore, func(i string) bool {
		i = strings.ToLower(i)
		return k == i || strings.HasPrefix(k, i+delimiter)
	})
}

// expandValue expands all the strings in v, including those in lists
// and maps, using opts. Values of all other types are returned
// unchanged.
func expandValue(c *Config, opts *expandOptions, v any) any {
	switch t := v.(type) {
	case string:
		return expandString(c, opts, t)
	case []string:
		return expandStrings(c, opts, t)
	case []any:
		nv := make([]any, len(t))
		for i, e := range t {
			nv[i] = expandValue(c, opts, e)
		}
		return nv
	case map[string]any:
		nv := make(map[string]any, len(t))
		for k, e := range t {
			nv[k] = expandValue(c, opts, e)
		}
		return nv
	case map[string]string:
		nv := make(map[string]string, len(t))
		for k, e := range t {
			nv[k] = expandString(c, opts, e)
		}
		return nv
	}
	return v
}
