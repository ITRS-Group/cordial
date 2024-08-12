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
	"path"
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

// LatestVersion returns the name of the latest release for component
// type ct on host h. The comparison is done using semantic versioning
// and any metadata is checked against the host platform_id - if they do
// not match then it is not latest - unless the prefix contains it. The
// matching is limited by the optional prefix filter. An error is
// returned if there are problems accessing the directories or parsing
// any names as semantic versions.
func LatestVersion(h *Host, ct *Component, versionPrefix string) (latest string, err error) {
	dir := h.PathTo("packages", ct.String())
	ents, err := h.ReadDir(dir)
	if err != nil {
		return
	}

	semver, _ := version.NewVersion("0.0.0")
	platformid := getPlatformId(h.GetString(h.Join("osinfo", "platform_id")))

	for _, ent := range ents {
		if !ent.IsDir() {
			continue
		}
		if versionPrefix != "" {
			if !strings.HasPrefix(ent.Name(), versionPrefix) {
				continue
			}
		}

		sv, err := version.NewVersion(strings.TrimLeftFunc(ent.Name(), func(r rune) bool { return !unicode.IsNumber(r) }))
		if err != nil {
			return latest, err
		}
		meta := sv.Metadata()
		if meta != "" && meta != platformid {
			continue
		}
		if sv.LessThan(semver) {
			continue
		}
		semver = sv
		latest = semver.Original()
	}

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

	v1p := strings.FieldsFunc(version1, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	if len(v1p) == 0 {
		v1p = []string{"GA"}
	} else if v1p[0] == "" {
		v1p[0] = "GA"
	} else {
		version1 = strings.TrimPrefix(version1, v1p[0])
	}
	v1, err := version.NewVersion(version1)
	if err != nil {
		// if version1 is unparseable, treat version2 as greater
		return 1
	}
	v2p := strings.FieldsFunc(version2, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	if len(v2p) == 0 {
		v2p = []string{"GA"}
	} else if v2p[0] == "" {
		v2p[0] = "GA"
	} else {
		version2 = strings.TrimPrefix(version2, v2p[0])
	}
	v2, err := version.NewVersion(version2)
	if err != nil {
		// if version2 is unparseable, treat version2 as greater
		return -1
	}
	if i := v1.Compare(v2); i != 0 {
		return i
	}
	return strings.Compare(v1p[0], v2p[0])
}
