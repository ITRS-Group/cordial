/*
Copyright © 2022 ITRS Group

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
	"bytes"
	"path"
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial/pkg/host"
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

		final = p
		if i == 0 || paths[0] == "" {
			return
		}
		if dir == "" {
			if err = r.Rename(p, paths[0]); err != nil {
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
				return
			}
		}

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

	return
}

// AbbreviateHome replaces the user's home directory prefix on path p
// with `~/` and cleans the result. If the home directory is not
// present, or there is an error resolving it, then a cleaned copy of
// the original string is returned. If the result is am empty string,
// return "."
func AbbreviateHome(p string) string {
	home, err := UserHomeDir()
	if err != nil {
		return path.Clean(p)
	}
	if strings.HasPrefix(p, home) {
		return filepath.ToSlash("~" + strings.TrimPrefix(p, home))
	}
	return path.Clean(p)
}

// ExpandHome replaces a leading `~/` on p with the user's home
// directory. If p does not have a prefix of `~/` then a cleaned copy is
// returned. If there is an error resolving the user's home directory
// then the path is returned relative to the root directory, i.e. with
// just the `~` removed (and cleaned).
func ExpandHome(p string) string {
	if !strings.HasPrefix(p, "~/") {
		return p
	}
	home, err := UserHomeDir()
	if err != nil {
		return strings.TrimPrefix(p, "~")
	}
	return path.Join(home, strings.TrimPrefix(p, "~"))
}

var (
	homePrefix         = "~"
	homePrefixDir      = "~/"
	homePrefixBytes    = []byte(homePrefix)
	homePrefixDirBytes = []byte(homePrefixDir)
)

// ExpandHomeBytes replaces a leading `~/` on p with the user's home
// directory. If p does not have a prefix of `~/` then a cleaned copy is
// returned. If there is an error resolving the user's home directory
// then the path is returned relative to the root directory, i.e. with
// just the `~` removed (and cleaned).
func ExpandHomeBytes(p []byte) []byte {
	if !bytes.HasPrefix(p, homePrefixDirBytes) {
		return p
	}
	home, err := UserHomeDir()
	if err != nil {
		return bytes.TrimPrefix(p, homePrefixBytes)
	}
	return []byte(path.Join(home, strings.TrimPrefix(string(p), homePrefix)))
}
