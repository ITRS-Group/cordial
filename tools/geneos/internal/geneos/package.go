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

package geneos

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/hashicorp/go-version"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/rs/zerolog/log"
)

// list of platform in release package names
var platformToMetaList = []string{
	"el8",
}

type ReleaseDetails struct {
	Component string    `json:"Component"`
	Host      string    `json:"Host"`
	Version   string    `json:"Version"`
	Latest    bool      `json:"Latest,string"`
	Links     []string  `json:"Links,omitempty"`
	ModTime   time.Time `json:"LastModified"`
	Path      string    `json:"Path"`
}

type Releases []ReleaseDetails

func (r Releases) Len() int {
	return len(r)
}

func (r Releases) Less(i, j int) bool {
	vi, _ := version.NewVersion(r[i].Version)
	vj, _ := version.NewVersion(r[j].Version)
	return vi.LessThan(vj)
}

func (r Releases) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

// GetReleases returns a slice of PackageDetails containing all the
// directories Geneos packages directory on the given host. Symlinks in
// the packages directory are matches to any targets and unmatched
// symlinks are ignored.
//
// No validation is done on the contents, only that a directory exists.
func GetReleases(h *Host, ct *Component) (releases Releases, err error) {
	if !h.IsAvailable() {
		return nil, host.ErrNotAvailable
	}
	basedir := h.Filepath("packages", ct)
	ents, err := h.ReadDir(basedir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return
	}

	var links = make(map[string][]string)

	for _, ent := range ents {
		einfo, err := ent.Info()
		if err != nil {
			// skip entries with errors
			log.Debug().Err(err).Msg("skipping")
			continue
		}
		if einfo.Mode()&fs.ModeSymlink != 0 {
			link, err := h.Readlink(filepath.Join(basedir, ent.Name()))
			if err != nil {
				// skip entries with errors
				log.Debug().Err(err).Msg("skipping")
				continue
			}
			links[link] = append(links[link], ent.Name())
		}
	}

	latest, _ := LatestVersion(h, ct, "")
	for _, ent := range ents {
		if ent.IsDir() {
			einfo, err := ent.Info()
			if err != nil {
				// skip entries with errors
				log.Debug().Err(err).Msg("skipping")
				continue
			}
			links := links[ent.Name()]
			mtime := einfo.ModTime().UTC()
			releases = append(releases, ReleaseDetails{
				Component: ct.String(),
				Host:      h.String(),
				Version:   ent.Name(),
				Latest:    ent.Name() == latest,
				Links:     links,
				ModTime:   mtime,
				Path:      filepath.Join(basedir, ent.Name()),
			})
		}
	}

	sort.Sort(releases)

	return releases, nil
}

func getVersions(r *Host, ct *Component) (versions map[string]*version.Version, originals map[string]string) {
	dir := r.Filepath("packages", ct.String())
	ents, err := r.ReadDir(dir)
	if err != nil {
		return
	}

	i := 0
	for _, d := range ents {
		if d.IsDir() {
			ents[i] = d
			i++
		}
	}
	ents = ents[:i]

	versions = make(map[string]*version.Version)
	originals = make(map[string]string, len(ents)) // map processed to original entry

	for _, d := range ents {
		n := d.Name()
		v1p := strings.FieldsFunc(n, func(r rune) bool {
			return !unicode.IsLetter(r)
		})
		originals[n] = n
		if len(v1p) > 0 && v1p[0] != "" {
			p := strings.TrimPrefix(n, v1p[0])
			originals[p] = n
			n = p
		}
		v1, err := version.NewVersion(n)
		if err == nil { // valid version
			if v1.Metadata() != "" {
				delete(versions, v1.Core().String())
			}
			versions[n] = v1
		}
	}
	return
}

func adjacentVersions(r *Host, ct *Component, current string) (prev string, next string, err error) {
	if current == "" {
		log.Debug().Msg("current version must be set, ignoring")
		return
	}
	cv, err := version.NewVersion(current)
	if err != nil {
		log.Debug().Err(err).Msgf("unable to parse version '%s', ignoring", current)
		return
	}

	versions, originals := getVersions(r, ct)
	if len(versions) == 0 {
		return "", "", nil
	}
	prevVers := []*version.Version{}
	for _, v := range versions {
		if cv.GreaterThan(v) {
			prevVers = append(prevVers, v)
		}
	}
	sort.Sort(version.Collection(prevVers))
	if len(prevVers) > 0 {
		prev = originals[prevVers[len(prevVers)-1].Original()]
	}
	NextVers := []*version.Version{}
	for _, v := range versions {
		if cv.LessThan(v) {
			NextVers = append(NextVers, v)
		}
	}
	sort.Sort(version.Collection(NextVers))
	if len(NextVers) > 0 {
		next = originals[NextVers[0].Original()]
	}
	return
}

// previousVersion returns the latest installed package that is earlier than version current
func previousVersion(r *Host, ct *Component, current string) (prev string, err error) {
	if current == "" {
		log.Debug().Msg("current version must be set, ignoring")
		return
	}
	cv, err := version.NewVersion(current)
	if err != nil {
		log.Debug().Err(err).Msgf("unable to parse version '%s', ignoring", current)
		return
	}

	versions, originals := getVersions(r, ct)
	if len(versions) == 0 {
		return "", nil
	}
	vers := []*version.Version{}
	for _, v := range versions {
		if cv.GreaterThan(v) {
			vers = append(vers, v)
		}
	}
	sort.Sort(version.Collection(vers))
	if len(vers) > 0 {
		prev = originals[vers[len(vers)-1].Original()]
	}
	return
}

func nextVersion(r *Host, ct *Component, current string) (next string, err error) {
	if current == "" {
		log.Debug().Msg("current version must be set, ignoring")
		return
	}
	cv, err := version.NewVersion(current)
	if err != nil {
		log.Debug().Err(err).Msgf("unable to parse version '%s', ignoring", current)
		return
	}

	versions, originals := getVersions(r, ct)
	if len(versions) == 0 {
		return "", nil
	}

	vers := []*version.Version{}
	for _, v := range versions {
		if cv.LessThan(v) {
			vers = append(vers, v)
		}
	}
	sort.Sort(version.Collection(vers))
	if len(vers) > 0 {
		next = originals[vers[0].Original()]
	}
	return
}

// LatestVersion returns the name of the latest release for component
// type ct on host h. The comparison is done using semantic versioning
// and any metadata is ignored. The matching is limited by the optional
// prefix filter. An error is returned if there are problems accessing
// the directories or parsing any names as semantic versions.
func LatestVersion(r *Host, ct *Component, prefix string) (v string, err error) {
	dir := r.Filepath("packages", ct.String())
	dirs, err := r.ReadDir(dir)
	if err != nil {
		return
	}

	semver, _ := version.NewVersion("0.0.0")

	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		if prefix != "" {
			if !strings.HasPrefix(d.Name(), prefix) {
				continue
			}
		}

		sv, err := version.NewVersion(d.Name())
		if err != nil {
			return v, err
		}
		if sv.LessThan(semver) {
			continue
		}
		semver = sv
		v = semver.Original()
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

// Install a Geneos software release. The destination host h and the
// component type ct must be given. options controls behaviour like
// local only and restarts of affected instances.
func Install(h *Host, ct *Component, options ...Options) (err error) {
	if h == ALL || ct == nil {
		return ErrInvalidArgs
	}

	options = append(options, PlatformID(h.GetString(h.Join("osinfo", "platform_id"))))

	opts := EvalOptions(options...)

	// open and unarchive if given a tar.gz

	archive, filename, err := openArchive(ct, options...)
	if err != nil {
		return
	}
	defer archive.Close()

	if dir, err := unarchive(h, ct, archive, filename, options...); err != nil {
		if errors.Is(err, fs.ErrExist) {
			fmt.Printf("%s on %s version %q already exists as %q\n", ct, h, opts.version, dir)
			return err
		}
		return err
	}

	if opts.doupdate {
		Update(h, ct, options...)
	}
	return
}

// split an package archive name into type and version
var archiveRE = regexp.MustCompile(`^geneos-(web-server|fixanalyser2-netprobe|file-agent|\w+)-([\w\.-]+?)[\.-]?linux`)

// filenameToComponent transforms an archive filename and returns the
// component and version or an error if the file format is not
// recognised
func filenameToComponent(filename string) (ct *Component, version string, err error) {
	parts := archiveRE.FindStringSubmatch(filename)
	if len(parts) != 3 {
		err = fmt.Errorf("%q: %w", filename, ErrInvalidArgs)
		return
	}
	version = parts[2]
	// replace '-' prefix of recognised platform suffixes with '+' so work with semver as metadata
	for _, m := range platformToMetaList {
		version = strings.ReplaceAll(version, "-"+m, "+"+m)
	}

	ct = FindComponent(parts[1])
	return
}
