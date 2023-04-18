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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/hashicorp/go-version"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/tools/geneos/internal/host"
)

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
func GetReleases(h *host.Host, ct *Component) (releases Releases, err error) {
	if !h.Exists() {
		return nil, fmt.Errorf("host does not exist")
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

	latest, _ := LatestRelease(h, basedir, "", func(d os.DirEntry) bool { // ignore error, empty is valid
		return !d.IsDir()
	})

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

// LatestRelease returns the latest release version named by a
// sub-directory in dir, on host r based on semver. A prefix filter can
// be used to limit matches and a filter function to further refine
// matches.
//
// If there is semver metadata, check for platform_id on host and
// remove any non-platform metadata from list before sorting
func LatestRelease(r *host.Host, dir, prefix string, filter func(os.DirEntry) bool) (latest string, err error) {
	dirs, err := r.ReadDir(dir)
	if err != nil {
		return
	}

	newdirs := dirs[:0]
	for _, d := range dirs {
		if strings.HasPrefix(d.Name(), prefix) {
			newdirs = append(newdirs, d)
		}
	}
	dirs = newdirs

	var versions = make(map[string]*version.Version)
	var originals = make(map[string]string, len(dirs)) // map of processed to original entries

	for _, d := range dirs {
		if filter != nil && filter(d) {
			continue
		}
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
	if len(versions) == 0 {
		return "", nil
	}
	vers := []*version.Version{}
	for _, v := range versions {
		vers = append(vers, v)
	}
	sort.Sort(version.Collection(vers))
	return originals[vers[len(vers)-1].Original()], nil
}

func getVersions(r *host.Host, ct *Component) (versions map[string]*version.Version, originals map[string]string) {
	dir := r.Filepath("packages", ct.String())
	dirs, err := r.ReadDir(dir)
	if err != nil {
		return
	}

	newdirs := dirs[:0]
	for _, d := range dirs {
		if d.IsDir() { // only subdirs, ignore files and links
			newdirs = append(newdirs, d)
		}
	}
	dirs = newdirs

	versions = make(map[string]*version.Version)
	originals = make(map[string]string, len(dirs)) // map processed to original entry

	for _, d := range dirs {
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

func AdjacentVersions(r *host.Host, ct *Component, current string) (prev string, next string, err error) {
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

// PreviousVersion returns the latest installed package that is earlier than version current
func PreviousVersion(r *host.Host, ct *Component, current string) (prev string, err error) {
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

func NextVersion(r *host.Host, ct *Component, current string) (next string, err error) {
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
// and any metadata is ignored. An error is returned if there are
// problems accessing the directories or parsing any names as semantic
// versions.
func LatestVersion(r *host.Host, ct *Component) (v string, err error) {
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
		n := d.Name()

		sv, err := version.NewVersion(n)
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
func CompareVersion(version1, version2 string) int {
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
		panic(err)
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
		panic(err)
	}
	if i := v1.Compare(v2); i != 0 {
		return i
	}
	return strings.Compare(v1p[0], v2p[0])
}

// Install installs a Geneos software release. The host must be given
// and call be 'all'. If a component type is passed in ct then only that
// component release is installed.
func Install(h *host.Host, ct *Component, options ...GeneosOptions) (err error) {
	if h == host.ALL {
		return ErrInvalidArgs
	}

	if ct == nil {
		for _, ct := range RealComponents() {
			if err = Install(h, ct, options...); err != nil {
				if errors.Is(err, fs.ErrExist) {
					continue
				}
				return
			}
		}
		return nil
	}

	options = append(options, PlatformID(h.GetString("osinfo.platform_id")))

	opts := EvalOptions(options...)

	reader, filename, err := openArchive(ct, options...)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err = unarchive(h, ct, filename, reader, options...); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil
		}
		return err
	}

	if opts.doupdate {
		Update(h, ct, options...)
	}
	return
}
