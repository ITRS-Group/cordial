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

package geneos

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"slices"
	"sort"
	"strings"
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
	vi, err := version.NewVersion(strings.TrimLeftFunc(r[i].Version, func(r rune) bool { return !unicode.IsNumber(r) }))
	if err != nil {
		log.Debug().Err(err).Msg("")
	}
	vj, err := version.NewVersion(strings.TrimLeftFunc(r[j].Version, func(r rune) bool { return !unicode.IsNumber(r) }))
	if err != nil {
		log.Debug().Err(err).Msg("")
	}
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
	ents, err := h.ReadDir(basedir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return
	}

	var links = make(map[string][]string)
	for _, dir := range ents {
		if dir.Type()&fs.ModeSymlink != 0 {
			if link, err := h.Readlink(path.Join(basedir, dir.Name())); err == nil {
				links[link] = append(links[link], dir.Name())
			}
		}
	}

	latest := ""
	versions, err := InstalledReleases(h, ct)
	if err != nil {
		// not found is ok, just return an empty slice
		if errors.Is(err, fs.ErrNotExist) {
			err = nil
		}
		return
	}
	if len(versions) > 0 {
		latest = versions[len(versions)-1]
	}

	// latest, _ := LatestInstalledVersion(h, ct, "")
	for _, ent := range ents {
		if ent.IsDir() {
			info, err := ent.Info()
			if err != nil {
				// skip entries with errors
				log.Debug().Err(err).Msg("skipping")
				continue
			}
			releases = append(releases, ReleaseDetails{
				Component: ct.String(),
				Host:      h.String(),
				Version:   ent.Name(),
				Latest:    ent.Name() == latest,
				Links:     links[ent.Name()],
				ModTime:   info.ModTime().UTC(),
				Path:      path.Join(basedir, ent.Name()),
			})
		}
	}

	sort.Sort(releases)

	return releases, nil
}

// Install a Geneos software release. The destination host h and the
// component type ct must be given. options controls behaviour like
// local only and restarts of affected instances.
func Install(h *Host, ct *Component, options ...PackageOptions) (err error) {
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
		return nil
	}

	options = append(options, SetPlatformID(h.GetString(h.Join("osinfo", "platform_id"))))

	opts := evalOptions(options...)

	// open an unarchive if given a tar.gz
	archive, filename, err := openArchive(ct, options...)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			var dir bool
			if opts.localArchive != "" {
				u, _ := url.Parse(opts.localArchive)
				if u.Scheme == "https" || u.Scheme == "http" {
					return
				}
				if s, err := h.Stat(opts.localArchive); err == nil && s.IsDir() {
					dir = true
				}
			}
			if opts.localOnly || (!opts.downloadonly && dir) {
				log.Debug().Msgf("%s not found at/in %s but local install only selected, skipping", ct, opts.localArchive)
				return nil
			}
		}
		return
	}
	defer archive.Close()

	if opts.downloadonly {
		return
	}

	if dest, err := unarchive(h, ct, archive, filename, options...); err != nil {
		if errors.Is(err, fs.ErrExist) {
			log.Debug().Msgf("%s on %s already installed as %q\n", ct, h, dest)
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
func Update(h *Host, ct *Component, options ...PackageOptions) (err error) {
	// before updating a specific type on a specific host, loop
	// through related types, hosts and components. continue to
	// other items if a single update fails?
	//

	for _, h := range h.OrList() {
		for _, ct := range ct.OrList() {
			if err = update(h, ct, options...); err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					fmt.Println(err)
				}
			}
		}
	}

	return nil
}

// update is the core function and must be called with non-wild ct and
// host
func update(h *Host, ct *Component, options ...PackageOptions) (err error) {
	if ct == nil || h == ALL {
		return ErrInvalidArgs
	}

	// update each associated package type for a parent component
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

	version := ""
	versions, err := InstalledReleases(h, ct)
	if err != nil {
		// not found is ok, just return an empty slice
		if errors.Is(err, fs.ErrNotExist) {
			err = nil
		}
		return
	}
	versions = slices.DeleteFunc(versions, func(version string) bool { return !strings.HasPrefix(version, opts.version) })
	if len(versions) > 0 {
		version = versions[len(versions)-1]
	}

	if version == "" {
		return fmt.Errorf("%s version %q on %s: %w", ct, originalVersion, h, os.ErrNotExist)
	}

	// does the version directory exist?
	existing, err := h.Readlink(basepath)
	if err != nil {
		log.Debug().Msgf("cannot read link for existing version %s", basepath)
	}

	// before removing existing link, check there is something to link to
	if _, err = h.Stat(path.Join(basedir, version)); err != nil {
		return fmt.Errorf("cannot update %s on %s using filter %q: %w", ct, h, opts.version, os.ErrNotExist)
	}

	// if we get here from a package install then that will have already
	// been filtered for "force" in the caller
	if existing == version {
		log.Debug().Msgf("existing == version %s", version)
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
	if err = h.Symlink(version, basepath); err != nil {
		return err
	}
	fmt.Printf("%s %q on %s updated to %s\n", ct, path.Base(basepath), h, version)
	return nil
}
