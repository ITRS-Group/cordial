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
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"unicode"

	"github.com/hashicorp/go-version"
)

// CurrentVersion returns the version that base points to for the
// component ct on host h. If base is not a symlink then it is returned
// unchanged. Returns version set to "unknown" on error.
func CurrentVersion(h *Host, ct *Component, base string) (version string, err error) {
	var i int

	dir := h.PathTo("packages", ct.String())
	version = base

	// follow the symlink up to 10 times, if it is a symlink and return
	// as soon as it is not a link.
	for i = 0; i < 10; i++ {
		var lversion string
		basepath := h.Join(dir, version)
		_, err = h.Lstat(basepath)
		if err != nil {
			version = "unknown"
			return
		}

		// return if it is not a symlink
		lversion, err = h.Readlink(basepath)
		if err != nil {
			// if we can't readlink then it is not a symlink, return the
			// last known version a symlink did point to
			err = nil
			return
		}
		version = lversion

		if h.IsAbs(version) {
			// if the version is absolute then we need to make it relative to the
			// directory we are in so that it can be used as a relative path
			// in the config file.
			version, err = h.Rel(dir, version)
			if err != nil {
				log.Debug("relative path", slog.Any("error", err))
			}
		}

		if version == base {
			err = syscall.ELOOP
			log.Debug("loop", slog.Any("error", err))
			version = "unknown"
			return
		}
	}
	if i == 10 {
		err = fmt.Errorf("too many levels of symbolic link (>10)")
		log.Debug("levels", slog.Any("error", err))
		version = "unknown"
	}

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

	// remove non-directories and incomplete installs
	ents = slices.DeleteFunc(ents, func(d fs.DirEntry) bool {
		if IsStagingDir(d.Name()) {
			return true
		}
		st, err := h.Lstat(h.Join(dir, d.Name()))
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
func LocalArchives(ct *Component, options ...GeneosOption) (archives []string, err error) {
	opts := evalOptions(options...)

	if !IsDir(opts.source) {
		err = fmt.Errorf("source %q is not a directory", opts.source)
		return
	}

	dir := opts.source

	log.Debug("opening local archive directory", slog.String("directory", dir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// remove hidden (including in-progress downloads) and non-files
	entries = slices.DeleteFunc(entries, func(d fs.DirEntry) bool {
		if strings.HasPrefix(d.Name(), ".") {
			return true
		}
		st, err := os.Lstat(filepath.Join(dir, d.Name()))
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

	log.Debug("archives before platform filter", slog.Any("archives", archives))

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

	log.Debug("archives after platform filter", slog.Any("archives", archives))

	return
}

// CompareVersion takes two Geneos package versions and returns an int
// that is 0 if they are identical, negative if version1 < version2 and
// positive is version1 > version2. If the version is prefixed with non
// numeric values then "GA" is always greater than "RA" (general versus
// restricted availability) for the same numeric version, otherwise a
// lexical comparison is done on the prefixes. A version without a
// prefix is treated as "GA".
//
// If either version is empty or unparseable then the return value is
// set to favour the other version - or 0 if both are empty strings.
// Two unparseable versions are compared lexically so that the result is
// always consistent, i.e. CompareVersion(a, b) == -CompareVersion(b, a).
func CompareVersion(version1, version2 string) int {
	p1, v1 := parseReleaseVersion(version1)
	p2, v2 := parseReleaseVersion(version2)

	switch {
	case v1 == nil && v2 == nil:
		return strings.Compare(version1, version2)
	case v1 == nil:
		return -1
	case v2 == nil:
		return 1
	}

	if i := v1.Compare(v2); i != 0 {
		return i
	}

	return compareVersionPrefix(p1, p2)
}

// parseReleaseVersion splits version into any leading non-numeric
// prefix, which defaults to "GA", and the parsed numeric version. If
// version is empty or cannot be parsed then the returned version is
// nil.
func parseReleaseVersion(v string) (prefix string, parsed *version.Version) {
	i := strings.IndexFunc(v, unicode.IsDigit)
	if i < 0 {
		return "", nil
	}
	prefix = v[:i]
	if prefix == "" {
		prefix = "GA"
	}
	parsed, err := version.NewVersion(v[i:])
	if err != nil {
		return "", nil
	}
	return
}

// compareVersionPrefix orders release prefixes for the same numeric
// version: "GA" is greater than "RA" and anything else is compared
// lexically.
func compareVersionPrefix(a, b string) int {
	a, b = strings.ToUpper(a), strings.ToUpper(b)
	switch {
	case a == b:
		return 0
	case a == "GA" && b == "RA":
		return 1
	case a == "RA" && b == "GA":
		return -1
	}
	return strings.Compare(a, b)
}

const stagingPrefix = ".installing-"

// StagingDir returns the hidden sibling directory used to unpack a
// release before it is renamed to dir.
func StagingDir(h *Host, dir string) string {
	return h.Join(h.Dir(dir), stagingPrefix+h.Base(dir))
}

// IsStagingDir reports whether name is a release staging directory,
// which may be left behind by an interrupted install.
func IsStagingDir(name string) bool {
	return strings.HasPrefix(name, stagingPrefix)
}
