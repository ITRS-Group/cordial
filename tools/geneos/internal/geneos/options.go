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
	"io"
	"path"
	"slices"

	"github.com/itrs-group/cordial/pkg/config"
)

// geneosOptions defines the internal options for various operations in
// the geneos package
type geneosOptions struct {
	archive               io.ReadCloser
	archivename           string
	archivesize           int64
	basename              string
	components            []*Component
	doupdate              bool
	downloadbase          string
	downloadonly          bool
	downloadtype          string
	geneosdir             string
	headers               []string
	host                  *Host
	insecure              bool
	localOnly             bool
	nosave                bool
	override              bool
	overrideVersion       string
	password              config.Secret
	platformId            string
	promptRootDir         func(string, ...any) (string, error)
	restart               []Instance
	signingBundle         string
	signingBundlePassword config.Secret
	signingBundlePrompt   func(bool, int, ...string) (config.Secret, error)
	source                string
	start                 func(Instance, ...any) error
	stop                  func(Instance, ...any) error
	username              string
	version               string
}

// GeneosOption can be passed to various function and influence
// behaviour related to download, unpacking and using release packages
type GeneosOption func(*geneosOptions)

func evalOptions(options ...GeneosOption) (d *geneosOptions) {
	// defaults
	d = &geneosOptions{
		downloadbase: "releases",
		downloadtype: "resources",
		source:       path.Join(LocalRoot(), "packages", "downloads"),
		host:         LOCAL,
	}
	for _, opt := range options {
		opt(d)
	}

	// make sure the components are unique
	slices.SortFunc(d.components, func(a, b *Component) int {
		if a.Name < b.Name {
			return -1
		} else if a.Name > b.Name {
			return 1
		}
		return 0
	})
	d.components = slices.Compact(d.components)

	// only check username if the source is a valid URL, and not a file
	if IsURL(d.source) && d.username == "" {
		cr := config.FindCreds(d.source)
		if cr != nil {
			d.username = config.Get[string](cr, "username")
			d.password = config.Get[config.Secret](cr, "password")
		}
	}
	return
}

func ForComponent(component *Component) GeneosOption {
	return func(d *geneosOptions) {
		if component == nil {
			return
		}
		d.components = append(d.components, component)
	}
}

func Insecure(insecure bool) GeneosOption {
	return func(d *geneosOptions) {
		d.insecure = insecure
	}
}

func SigningBundle(signingBundle string) GeneosOption {
	return func(d *geneosOptions) {
		d.signingBundle = signingBundle
	}
}

func SigningBundlePassword(signingBundlePassword config.Secret) GeneosOption {
	return func(d *geneosOptions) {
		d.signingBundlePassword = signingBundlePassword
	}
}

func SigningBundlePrompt(prompt func(bool, int, ...string) (config.Secret, error)) GeneosOption {
	return func(d *geneosOptions) {
		d.signingBundlePrompt = prompt
	}
}

func RootDirPrompt(prompt func(string, ...any) (string, error)) GeneosOption {
	return func(d *geneosOptions) {
		d.promptRootDir = prompt
	}
}

// DownloadOnly prevents the unarchiving of the selected packages.
// Downloads are stored in the directory given to Source() or the
// default `packages/download` directory. NoSave and DownloadOnly
// are mutually exclusive.
func DownloadOnly(download bool) GeneosOption {
	return func(d *geneosOptions) {
		d.downloadonly = download
	}
}

// Onto sets the destination host for the installation. This is
// used to determine the OS and architecture required for any download.
func Onto(host *Host) GeneosOption {
	return func(po *geneosOptions) {
		po.host = host
	}
}

// NoSave stops downloads from being saved in the archive directory.
// NoSave and DownloadOnly are mutually exclusive.
func NoSave(nosave bool) GeneosOption {
	return func(d *geneosOptions) {
		d.nosave = nosave
	}
}

// LocalOnly uses only existing archives and prevents attempts to
// download releases
func LocalOnly(local bool) GeneosOption {
	return func(d *geneosOptions) {
		d.localOnly = local
	}
}

// Source is the archive location or the specific release file. It can
// be a directory, in which case that directory is used for the
// appropriate archive file(s), a specific file or a URL to a release.
// If the file is a URL, it is downloaded to the local archive
// directory, which defaults to `packages/downloads`.
//
// If source is an empty string then a default value is used, pointing
// to the `package/download` directory in the geneos installation
// directory.
func Source(source string) GeneosOption {
	return func(d *geneosOptions) {
		if source == "" {
			d.source = path.Join(LocalRoot(), "packages", "downloads")
			return
		}
		d.source = source
	}
}

// Override ignores existing directories or files and also overrides
// protection for running instances in upgrades
func Override(force bool) GeneosOption {
	return func(d *geneosOptions) {
		d.override = force
	}
}

// OverrideVersion forces a specific version to be used and fails if not
// available
func OverrideVersion(version string) GeneosOption {
	return func(d *geneosOptions) {
		d.overrideVersion = version
	}
}

// Restart sets the instances to be restarted around the update
func Restart(instance ...Instance) GeneosOption {
	return func(d *geneosOptions) {
		d.restart = append(d.restart, instance...)
	}
}

// DoUpdate sets the option to also do an update after an install
func DoUpdate(update bool) GeneosOption {
	return func(d *geneosOptions) { d.doupdate = update }
}

// StartFunc sets the start function to call for each instance given in
// Restart(). It is required to avoid import loops.
func StartFunc(fn func(Instance, ...any) error) GeneosOption {
	return func(d *geneosOptions) {
		d.start = fn
	}
}

// StopFunc sets the start function to call for each instance given in
// Restart(). It is required to avoid import loops.
func StopFunc(fn func(Instance, ...any) error) GeneosOption {
	return func(d *geneosOptions) {
		d.stop = fn
	}
}

// Version sets the desired version number, defaults to "latest" in most
// cases. The version number is in the form `[GA]X.Y.Z` (or `RA` for
// snapshots)
func Version(version string) GeneosOption {
	return func(d *geneosOptions) {
		d.version = version
	}
}

// Basename sets the package binary basename, defaults to active_prod,
// for symlinks for update.
func Basename(basename string) GeneosOption {
	return func(d *geneosOptions) {
		d.basename = basename
	}
}

// SetPlatformID sets the (Linux) platform ID from the OS release info.
// Currently used to distinguish RHEL8/9 releases from others.
func SetPlatformID(platform string) GeneosOption {
	return func(d *geneosOptions) {
		d.platformId = platform
	}
}

// UseRoot sets the Geneos installation home directory (aka `geneos` in
// the settings)
func UseRoot(root string) GeneosOption {
	return func(d *geneosOptions) {
		d.geneosdir = root
	}
}

// Username is the remote access username for downloads
func Username(username string) GeneosOption {
	return func(d *geneosOptions) {
		d.username = username
	}
}

// Password is the remote access password for downloads
func Password(password config.Secret) GeneosOption {
	return func(d *geneosOptions) {
		d.password = password
	}
}

// UseNexus sets the flag to use nexus.itrsgroup.com for internal
// downloads instead of the default download URL in the settings. This
// also influences the way the remote path is searched and build, not
// just the base URL.
func UseNexus(yes bool) GeneosOption {
	return func(d *geneosOptions) {
		if yes {
			d.downloadtype = "nexus"
		} else {
			d.downloadtype = "resources"
		}
	}
}

// UseNexusSnapshots set the flag to use Nexus Snapshots rather than
// Releases.
func UseNexusSnapshots(yes bool) GeneosOption {
	return func(d *geneosOptions) {
		if yes {
			d.downloadbase = "snapshots"
		} else {
			d.downloadbase = "releases"
		}
	}
}

// Headers adds each arg as an HTTP header in the form NAME=VALUE for
// passing to URL requests. The NAME=VALUE is decomposed just before the
// request so that the order the headers are added as options is
// preserved. A header in the wrong format is ignored.
func Headers(header ...string) GeneosOption {
	return func(d *geneosOptions) {
		d.headers = append(d.headers, header...)
	}
}

// UseArchive sets the archive to use for installation. This can be used
// after calling OpenArchive() to test availability of the archive and
// then pass the archive to the installation function. If the archive is
// nil then it is ignored and the source option is used instead.
func UseArchive(archive io.ReadCloser, name string, size int64) GeneosOption {
	return func(d *geneosOptions) {
		d.archive = archive
		d.archivename = name
		d.archivesize = size
	}
}
