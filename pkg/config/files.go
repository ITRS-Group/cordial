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

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/host"
)

// MigrateFile iterates over newPath and oldPaths and finds the first
// regular file that exists. If this is not newPath then the found file
// is renamed to the newPath. The resulting oldPath is returned.
//
// If the newPath is an empty string then no rename takes place and the
// path of the first existing regular file is returned. If the newPath
// is a directory then the file is moved into that directory through a
// rename operation and a file with the first matching basename of any
// other arguments is returned (this avoids the second call returning
// nothing). Any empty strings in oldPaths is ignored.
func MigrateFile(h host.Host, newPath string, oldPaths ...string) (dest string) {
	if newPath == "" {
		// iterate over newPaths and find the first existing file
		for _, oldPath := range oldPaths {
			if p, err := h.Stat(oldPath); err == nil && p.Mode().IsRegular() {
				return oldPath
			}
		}
		// else return an empty destination
		return
	}

	// check is newPath already exists
	if st, err := h.Stat(newPath); err == nil && st.Mode().IsRegular() {
		// newPath is a existing regular file, just return it
		log.Debug().Msgf("newPath %s already exists, returning", newPath)
		return newPath
	}

	for i, oldPath := range oldPaths {
		if oldPath == "" {
			// skip empty paths
			log.Debug().Msgf("skipping empty oldPath at %d", i)
			continue
		}

		if st, err := h.Stat(newPath); err == nil {
			switch {
			case st.IsDir():
				// if newPath is a directory, move the file oldPath into it
				dest = path.Join(newPath, path.Base(oldPath))

				// if destination already exists and is a regular file, return
				if st, err := h.Stat(dest); err == nil && st.Mode().IsRegular() {
					return
				}

				// else rename the oldPath to the new destination
				if err := h.Rename(oldPath, dest); err != nil {
					log.Debug().Err(err).Msgf("failed to rename %s to %s", oldPath, path.Join(newPath, path.Base(oldPath)))
					return ""
				}

				return
			case st.Mode().IsRegular():
				if err := h.Rename(oldPath, newPath); err == nil {
					return newPath
				}
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
