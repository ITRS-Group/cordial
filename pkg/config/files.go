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
	"path"
	"strings"

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/rs/zerolog/log"
)

// PromoteFile iterates over paths and finds the first regular file that
// exists. If this is not the first element in the paths slice then the
// found file is renamed to the path of the first element. The resulting
// final path is returned.
//
// If the first element of paths is an empty string then no rename takes
// place and the first existing file is returned. If the first element
// is a directory then the file is moved into that directory through a
// rename operation and a file with the first matching basename of any
// other arguments is returned (this avoids the second call returning
// nothing).
func PromoteFile(r host.Host, paths ...string) (final string) {
	log.Debug().Msgf("paths: %v", paths)
	if len(paths) == 0 {
		return
	}

	var dir string
	if paths[0] != "" {
		if p0, err := r.Stat(paths[0]); err == nil && p0.IsDir() {
			dir = paths[0]
		}
	}

	for i, p := range paths {
		var err error
		if p == "" {
			continue
		}
		if p, err := r.Stat(p); err != nil || !p.Mode().IsRegular() {
			continue
		}

		log.Debug().Msgf("here: %s", p)
		final = p
		if i == 0 || paths[0] == "" {
			log.Debug().Msgf("returning paths[0]")
			return
		}
		if dir == "" {
			if err = r.Rename(p, paths[0]); err != nil {
				log.Debug().Msgf("renaming path %s to path %s", p, paths[0])
				return
			}
			final = paths[0]
		} else {
			final = path.Join(dir, path.Base(p))
			// don't overwrite existing, return that
			if p, err := r.Stat(final); err == nil && p.Mode().IsRegular() {
				return final
			}
			if err = r.Rename(p, final); err != nil {
				log.Debug().Msgf("renaming path %s to dir %s", p, final)
				return
			}
		}

		log.Debug().Msgf("returning path %s", final)
		return
	}

	if final == "" && dir != "" {
		for _, p := range paths[1:] {
			check := path.Join(dir, path.Base(p))
			if p, err := r.Stat(check); err == nil && p.Mode().IsRegular() {
				return check
			}
		}
	}

	log.Debug().Msgf("returning path %s", final)
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
func (cf *Config) ReadRCConfig(r host.Host, p string, prefix string, aliases map[string]string) (err error) {
	data, err := r.ReadFile(p)
	if err != nil {
		return
	}
	log.Debug().Msgf("loading config from %q", p)

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
			if nk, ok := aliases[lk]; ok {
				cf.Set(nk, v)
			} else {
				cf.Set(lk, v)
			}
		} else {
			// set env var
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	if len(env) > 0 {
		cf.Set("env", env)
	}

	// label the type as an "rc" to make it easy to check later
	cf.Type = "rc"

	return
}
