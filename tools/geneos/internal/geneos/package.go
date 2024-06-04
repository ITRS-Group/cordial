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
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/hashicorp/go-version"
	"github.com/rs/zerolog/log"
)

// list of platform in release package names
var platformSuffixList = []string{
	"el8",
	"el9",
}

// ReleaseDetails is a set of values for a release
type ReleaseDetails struct {
	Component string    `json:"Component"`
	Host      string    `json:"Host"`
	Version   string    `json:"Version"`
	Latest    bool      `json:"Latest,string"`
	Links     []string  `json:"Links,omitempty"`
	ModTime   time.Time `json:"LastModified"`
	Path      string    `json:"Path"`
}

// Releases is a slice of ReleaseDetails, used for sorting ReleaseDetails
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
	var ok bool
	if ok, err = h.IsAvailable(); !ok {
		return
	}
	basedir := h.PathTo("packages", ct.String())
	dirs, err := h.ReadDir(basedir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return
	}

	var links = make(map[string][]string)
	for _, dir := range dirs {
		if dir.Type()&fs.ModeSymlink != 0 {
			if link, err := h.Readlink(path.Join(basedir, dir.Name())); err == nil {
				links[link] = append(links[link], dir.Name())
			}
		}
	}

	latest, _ := LatestVersion(h, ct, "")
	for _, dir := range dirs {
		if dir.IsDir() {
			info, err := dir.Info()
			if err != nil {
				// skip entries with errors
				log.Debug().Err(err).Msg("skipping")
				continue
			}
			releases = append(releases, ReleaseDetails{
				Component: ct.String(),
				Host:      h.String(),
				Version:   dir.Name(),
				Latest:    dir.Name() == latest,
				Links:     links[dir.Name()],
				ModTime:   info.ModTime().UTC(),
				Path:      path.Join(basedir, dir.Name()),
			})
		}
	}

	sort.Sort(releases)

	return releases, nil
}

func getVersions(r *Host, ct *Component) (versions map[string]*version.Version, originals map[string]string) {
	dir := r.PathTo("packages", ct.String())
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
func LatestVersion(h *Host, ct *Component, prefix string) (v string, err error) {
	dir := h.PathTo("packages", ct.String())
	dirs, err := h.ReadDir(dir)
	if err != nil {
		return
	}

	semver, _ := version.NewVersion("0.0.0")
	platformid := h.GetString("platform_id")

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
		meta := sv.Metadata()
		if meta != "" && meta != platformid && !strings.HasSuffix(prefix, "+"+meta) {
			continue
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
	log.Debug().Msgf("host %s, component %s", h, ct)
	if h == ALL || ct == nil {
		return ErrInvalidArgs
	}

	if len(ct.PackageTypes) > 0 {
		for _, ct := range ct.PackageTypes {
			if err = Install(h, ct, options...); err != nil {
				log.Error().Err(err).Msg("")
			}
		}
		return
	}

	options = append(options, PlatformID(h.GetString(h.Join("osinfo", "platform_id"))))

	opts := evalOptions(options...)

	// open and unarchive if given a tar.gz

	archive, filename, err := openArchive(ct, options...)
	if err != nil {
		if errors.Is(err, ErrNotExist) {
			var dir bool
			if opts.archive != "" {
				u, _ := url.Parse(opts.archive)
				if u.Scheme == "https" || u.Scheme == "http" {
					return
				}
				if s, err := h.Stat(opts.archive); err == nil && s.IsDir() {
					dir = true
				}
			}
			if opts.local || (!opts.downloadonly && dir) {
				fmt.Printf("%s archive not found but local install selected, skipping\n", ct)
				return nil
			}
		}
		return
	}
	defer archive.Close()

	if opts.downloadonly {
		return
	}

	if dir, err := unarchive(h, ct, archive, filename, options...); err != nil {
		if errors.Is(err, fs.ErrExist) {
			log.Debug().Msgf("%s on %s already installed as %q\n", ct, h, dir)
			if opts.doupdate {
				log.Debug().Msg("update true")
				return Update(h, ct, options...)
			}
			return nil
		}
		return err
	}

	if opts.doupdate {
		log.Debug().Msg("update true")
		return Update(h, ct, options...)
	}
	return
}

// Update will check and update the base link given in the options. If
// the base link exists then the force option must be used to update it,
// otherwise it is created as expected. When called from unarchive()
// this allows new installs to work without explicitly calling update.
func Update(h *Host, ct *Component, options ...Options) (err error) {
	// before updating a specific type on a specific host, loop
	// through related types, hosts and components. continue to
	// other items if a single update fails?
	//

	for _, h := range h.OrList() {
		for _, ct := range ct.OrList() {
			if err = update(h, ct, options...); err != nil {
				fmt.Println(err)
			}
		}
	}

	return nil
}

// update is the core function and must be called with non-wild ct and
// host
func update(h *Host, ct *Component, options ...Options) (err error) {
	if ct == nil || h == ALL {
		return ErrInvalidArgs
	}

	if len(ct.PackageTypes) > 0 {
		for _, ct := range ct.PackageTypes {
			if err = Update(h, ct, options...); err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Error().Err(err).Msg("")
			}
		}
		return nil
	}

	// from here hosts and component types must be specified

	opts := evalOptions(options...)

	if opts.version == "" {
		opts.version = "latest"
	}

	originalVersion := opts.version

	log.Debug().Msgf("checking and updating %s on %s %q to %q", ct, h, opts.basename, opts.version)

	basedir := h.PathTo("packages", ct.String()) // use the actual ct not the parent, if there is one
	basepath := path.Join(basedir, opts.basename)

	if opts.version == "latest" {
		opts.version = ""
	}

	opts.version, err = LatestVersion(h, ct, opts.version)
	if err != nil {
		log.Debug().Err(err).Msg("")
	}

	if opts.version == "" {
		return fmt.Errorf("%s version %q on %s: %w", ct, originalVersion, h, os.ErrNotExist)
	}

	// does the version directory exist?
	existing, err := h.Readlink(basepath)
	if err != nil {
		log.Debug().Msgf("cannot read link for existing version %s", basepath)
	}

	// before removing existing link, check there is something to link to
	if _, err = h.Stat(path.Join(basedir, opts.version)); err != nil {
		return fmt.Errorf("%q version of %s on %s: %w", opts.version, ct, h, os.ErrNotExist)
	}

	// if we get here from a package install then that will have already
	// been filtered for "force" in the caller
	if existing == opts.version {
		log.Debug().Msgf("existing == version %s", opts.version)
		return nil
	}

	if opts.start != nil && opts.stop != nil {
		for _, c := range opts.restart {
			// only stop selected instances using components on the host we are working on
			if c.Host() != h {
				continue
			}
			// check for plain type or package type
			if c.Type() != ct && c.Config().GetString("pkgtype") != ct.String() {
				continue
			}
			if err = opts.stop(c, opts.force, false); err == nil {
				// only restart instances that we stopped, regardless of success of install/update
				defer opts.start(c)
			}
		}
	}

	if err = h.Remove(basepath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err = h.Symlink(opts.version, basepath); err != nil {
		return err
	}
	fmt.Printf("%s %q on %s updated to %s\n", ct, path.Base(basepath), h, opts.version)
	return nil
}
