/*
Copyright © 2024 ITRS Group

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

package geneos

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"unicode"

	"github.com/hashicorp/go-version"
	"github.com/rs/zerolog/log"
)

// CurrentVersion returns the version that base points to for the
// component ct on host h. If base is not a symlink then it is returned
// unchanged. Returns version set to "unknown" on error.
func CurrentVersion(h *Host, ct *Component, base string) (version string, err error) {
	var st fs.FileInfo
	var i int

	dir := h.PathTo("packages", ct.String())
	version = base

	for i = 0; i < 10; i++ {
		basepath := path.Join(dir, version)
		st, err = h.Lstat(basepath)
		if err != nil {
			log.Debug().Err(err).Msg("Lstat")
			version = "unknown"
			return
		}
		if st.Mode()&fs.ModeSymlink == 0 {
			if !st.IsDir() {
				err = syscall.ENOTDIR
				log.Debug().Err(err).Msg("symlink?")
				version = "unknown"
				return
			}
			// version = st.Name()
			return
		}
		version, err = h.Readlink(basepath)
		if err != nil {
			log.Debug().Err(err).Msg("readlink")
			version = "unknown"
			return
		}
		if version == base {
			err = syscall.ELOOP
			log.Debug().Err(err).Msg("loop")
			version = "unknown"
			return
		}
	}
	if i == 10 {
		err = fmt.Errorf("too many levels of symbolic link (>10)")
		log.Debug().Err(err).Msg("levels")
		version = "unknown"
	}

	log.Debug().Msgf("return %s", version)

	return
}

// InstalledReleases returns a sorted slice of all the installed version
// of component ct on host h as strings. No filtering is done for
// platform ID etc. as these are already installed on theo host given.
func InstalledReleases(h *Host, ct *Component) (versions []string, err error) {
	if h == nil || h == ALL || ct == nil {
		err = ErrInvalidArgs
		return
	}
	dir := h.PathTo("packages", ct.String())
	ents, err := h.ReadDir(dir)
	if err != nil {
		return
	}

	// remove non-directories
	ents = slices.DeleteFunc(ents, func(d fs.DirEntry) bool {
		st, err := h.Lstat(filepath.Join(dir, d.Name()))
		return err != nil || !st.IsDir()
	})

	for _, ent := range ents {
		versions = append(versions, ent.Name())
	}

	slices.SortFunc(versions, CompareVersion)

	return
}

// LocalArchives returns a slice of file names for release archives for
// component ct in the location given by options or the default packages
// download directory. If a platform ID is set using options then
// include those otherwise only non-platform specific releases are
// included.
func LocalArchives(ct *Component, options ...PackageOptions) (archives []string, err error) {
	opts := evalOptions(options...)

	log.Debug().Msgf("opening local archive directory %q", opts.localArchive)
	entries, err := os.ReadDir(opts.localArchive)
	if err != nil {
		return
	}

	// remove non-files from the list
	entries = slices.DeleteFunc(entries, func(d fs.DirEntry) bool {
		st, err := os.Lstat(filepath.Join(opts.localArchive, d.Name()))
		return err != nil || !st.Mode().IsRegular()
	})

	for _, d := range entries {
		archives = append(archives, d.Name())
	}

	archives = slices.DeleteFunc(archives, func(n string) bool {
		check := ct.String()
		if ct.ParentType != nil && len(ct.PackageTypes) > 0 {
			check = ct.ParentType.String()
		}

		if ct.DownloadInfix != "" {
			check = ct.DownloadInfix
		}

		return !strings.Contains(n, check)
	})

	log.Debug().Msgf("archives before platform filter: %v", archives)

	if opts.platformId == "" {
		archives = slices.DeleteFunc(archives, func(n string) bool {
			for _, p := range platformSuffixList {
				if strings.Contains(n, "-"+p+"-") {
					return true
				}
			}
			return false
		})
	} else {
		platformID := getPlatformId(opts.platformId)
		archives = slices.DeleteFunc(archives, func(n string) bool {
			if strings.Contains(n, "-"+platformID+"-") {
				return false
			}
			for _, p := range platformSuffixList {
				if strings.Contains(n, "-"+p+"-") {
					return true
				}
			}
			return false
		})
	}

	slices.SortFunc(archives, func(a, b string) int {
		var ap, bp bool
		for _, p := range platformSuffixList {
			if strings.Contains(a, "-"+p+"-") {
				ap = true
			}
			if strings.Contains(b, "-"+p+"-") {
				bp = true
			}
		}

		switch {
		case ap && !bp:
			return 1
		case !ap && bp:
			return -1
		default:
			return strings.Compare(a, b)
		}
	})

	log.Debug().Msgf("archives after platform filter: %v", archives)

	return
}

// CompareVersion takes two Geneos package versions and returns an int
// that is 0 if they are identical, negative if version1 < version2 and
// positive is version1 > version2. If the version is prefixed with non
// numeric values then "GA" is always greater thn "RA" (general versus
// restricted availability) for the same numeric version, otherwise a
// lexical comparison is done on the prefixes.
//
// If either version is empty or unparseable then the return value is
// set to favour the other version - or 0 if both are empty strings.
func CompareVersion(version1, version2 string) int {
	// cope with empty versions
	if version1 == "" && version2 == "" {
		return 0
	}
	if version1 == "" {
		return -1
	}
	if version2 == "" {
		return 1
	}

	v1parts := strings.FieldsFunc(version1, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	if len(v1parts) == 0 {
		v1parts = []string{"GA"}
	} else if v1parts[0] == "" {
		v1parts[0] = "GA"
	} else {
		version1 = strings.TrimPrefix(version1, v1parts[0])
	}
	v1, err := version.NewVersion(version1)
	if err != nil {
		// if version1 is unparseable, treat version2 as greater
		return 1
	}

	v2parts := strings.FieldsFunc(version2, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	if len(v2parts) == 0 {
		v2parts = []string{"GA"}
	} else if v2parts[0] == "" {
		v2parts[0] = "GA"
	} else {
		version2 = strings.TrimPrefix(version2, v2parts[0])
	}
	v2, err := version.NewVersion(version2)
	if err != nil {
		// if version2 is unparseable, treat version2 as greater
		return -1
	}

	if i := v1.Compare(v2); i != 0 {
		return i
	}

	return strings.Compare(v1parts[0], v2parts[0])
}
