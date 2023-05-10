/*
Copyright © 2022 ITRS Group

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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/rs/zerolog/log"
)

// OpenPromoteFile searches paths on remote r for the first file to exist and be
// readable. If this is not the first element in paths it then renames/moves ths
// found file to the path in the first element in the slice (unless it's an
// empty string). It returns an io.ReadSeekCloser for the open file and the
// final path. If there is an error moving the file then the returned path is
// that of the originally opened file.
func OpenPromoteFile(r host.Host, paths ...string) (f io.ReadSeekCloser, final string) {
	var err error
	final = PromoteFile(r, paths...)
	if final != "" {
		f, err = r.Open(final)
		if err != nil {
			final = ""
		}
	}
	return
}

// PromoteFile iterates over paths and finds the first regular file that
// exists. If this is not the first element in the paths slice then the
// found file is renamed to the path of the first element. The resulting
// final path is returned.
//
// If the first element of paths is an empty string then no rename takes
// place and the first existing file is returned. If the first element
// is a directory then the file is moved into that directory through a
// rename operation.
func PromoteFile(r host.Host, paths ...string) (final string) {
	for i, path := range paths {
		var err error
		if path == "" {
			continue
		}
		if p, err := r.Stat(path); err != nil || !p.Mode().IsRegular() {
			continue
		}

		final = path
		if i == 0 || paths[0] == "" {
			return
		}
		if err = r.Rename(path, paths[0]); err != nil {
			return
		}
		final = paths[0]
		return
	}

	return
}

// ReadConfig reads a configuration file from remote r.
//
// The configuration file path is set so that rewriting the file works
// without change.
func ReadConfig(r host.Host, path string, options ...LoadOptions) (cf *Config, err error) {
	if !r.IsAvailable() {
		err = fmt.Errorf("cannot reach %s", r)
		return
	}
	cf = New()
	cf.SetFs(r.GetFs())
	cf.SetConfigFile(path)
	if err = cf.ReadInConfig(); err != nil {
		return
	}
	return
}

// ReadRCConfig reads an old-style, legacy Geneos "ctl" layout
// configuration file and sets values in cf corresponding to updated
// equivalents.
//
// All empty lines and those beginning with "#" comments are ignored.
//
// The rest of the lines are treated as `name=value` pairs and are
// processed as follows:
//
//   - If `name` is either `binsuffix` (case-insensitive) or
//     `prefix`+`name` then it saved as a config item. This is looked up
//     in the `aliases` map and if there is a match then this new name is
//     used.
//   - All other `name=value` entries are saved as environment variables
//     in the configuration for the instance under the `Env` key.
func (cf *Config) ReadRCConfig(r host.Host, path string, prefix string, aliases map[string]string) (err error) {
	data, err := r.ReadFile(path)
	if err != nil {
		return
	}
	log.Debug().Msgf("loading config from %q", path)

	confs := make(map[string]string)

	scanner := bufio.NewScanner(bytes.NewBuffer(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		s := strings.SplitN(line, "=", 2)
		if len(s) != 2 {
			return fmt.Errorf("invalid line (must be key=value) %q", line)
		}
		key, value := s[0], s[1]
		// trim double and single quotes and tabs and spaces from value
		value = strings.Trim(value, "\"' \t")
		confs[key] = value
	}

	var env []string
	for k, v := range confs {
		lk := strings.ToLower(k)
		if lk == "binsuffix" || strings.HasPrefix(lk, prefix) {
			nk, ok := aliases[lk]
			if !ok {
				nk = lk
			}
			cf.Set(nk, v)
		} else {
			// set env var
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	if len(env) > 0 {
		cf.Set("Env", env)
	}

	return
}

// WriteConfig writes the configuration c to a file on remote r.
// func (c *Config) WriteConfig(r host.Host) {

// }