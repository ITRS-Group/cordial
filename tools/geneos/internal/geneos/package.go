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
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/hashicorp/go-version"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
)

type PackageDetails struct {
	Component string    `json:"Component"`
	Host      string    `json:"Host"`
	Version   string    `json:"Version"`
	Latest    bool      `json:"Latest,string"`
	Link      string    `json:"Link,omitempty"`
	ModTime   time.Time `json:"LastModified"`
	Path      string    `json:"Path"`
}

// GetPackages returns a slice if PackageDetails listing all the
// directories on the given host's Geneos packages directory. Symlinks
// in the packages directory are matches to any targets and unmatched
// symlinks are ignored.
//
// No validation is done on the contents, only that a directory exists.
func GetPackages(h *host.Host, ct *Component) (versions []PackageDetails, err error) {
	basedir := h.Filepath("packages", ct)
	ents, err := h.ReadDir(basedir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return
	}

	var links = make(map[string]string)
	for _, ent := range ents {
		einfo, err := ent.Info()
		if err != nil {
			return versions, err
		}
		if einfo.Mode()&fs.ModeSymlink != 0 {
			link, err := h.Readlink(filepath.Join(basedir, ent.Name()))
			if err != nil {
				return versions, err
			}
			links[link] = ent.Name()
		}
	}

	latest, _ := LatestRelease(h, basedir, "", func(d os.DirEntry) bool { // ignore error, empty is valid
		return !d.IsDir()
	})

	for _, ent := range ents {
		if ent.IsDir() {
			einfo, err := ent.Info()
			if err != nil {
				return versions, err
			}
			link := links[ent.Name()]
			mtime := einfo.ModTime().UTC()
			versions = append(versions, PackageDetails{
				Component: ct.String(),
				Host:      h.String(),
				Version:   ent.Name(),
				Latest:    ent.Name() == latest,
				Link:      link,
				ModTime:   mtime,
				Path:      filepath.Join(basedir, ent.Name()),
			})

		}
	}
	return versions, nil
}

// LatestRelease returns the latest sub-directory in dir, on host r
// based on semver. A prefix filter can be used to limit matches and a
// filter function to further refine matches. If there is metadata,
// check for platform_id on host and remove non-platform from list
// before sorting
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
	var originals = make(map[string]string, len(dirs)) // map processed to original entry

	for _, d := range dirs {
		if filter(d) {
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
		for _, t := range RealComponents() {
			if err = Install(h, t, options...); err != nil {
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
