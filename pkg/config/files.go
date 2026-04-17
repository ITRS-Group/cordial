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
	"os"
	"os/user"
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
	if after, ok := strings.CutPrefix(p, home); ok {
		return filepath.ToSlash("~" + after)
	}
	return path.Clean(p)
}

const (
	homePrefixDir = "~/"
)

// ResolveHome replaces any leading `~/` on p with the user's home
// directory. If p does not have a prefix of `~/` then a cleaned copy is
// returned. If there is an error resolving the user's home directory
// then the path is returned relative to the working directory, i.e. with
// just the `~/` removed (and cleaned).
func ResolveHome(p string) string {
	p2 := string(p)
	if !strings.HasPrefix(p2, homePrefixDir) {
		return p
	}
	home, err := UserHomeDir()
	if err != nil {
		return strings.TrimPrefix(p2, homePrefixDir)
	}
	return path.Join(home, strings.TrimPrefix(p2, homePrefixDir))
}

// UserHomeDir returns the home directory for username, or if none given
// then the current user. This works around empty environments by
// falling back to looking up the user.
func UserHomeDir(username ...string) (home string, err error) {
	if len(username) == 0 {
		if home, err = os.UserHomeDir(); err == nil { // all ok
			return
		}
		u, err := user.Current()
		if err != nil {
			return home, err
		}
		return u.HomeDir, nil
	}
	u, err := user.Lookup(username[0])
	if err != nil {
		return
	}
	return u.HomeDir, nil
}
