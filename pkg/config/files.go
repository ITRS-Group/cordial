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
	"path"

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
