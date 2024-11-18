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
	"path"

	"github.com/itrs-group/cordial/pkg/config"
)

// packageOptions defines the internal options for various operations in
// the geneos package
type packageOptions struct {
	localArchive string
	basename     string
	doupdate     bool
	downloadbase string
	downloadonly bool
	downloadtype string
	force        bool
	geneosdir    string
	host         *Host
	localOnly    bool
	nosave       bool
	override     string
	password     *config.Plaintext
	platformId   string
	restart      []Instance
	start        func(Instance, ...any) error
	stop         func(Instance, bool, bool) error
	username     string
	version      string
}

// PackageOptions can be passed to various function and influence
// behaviour related to download, unpacking and using release packages
type PackageOptions func(*packageOptions)

func evalOptions(options ...PackageOptions) (d *packageOptions) {
	// defaults
	d = &packageOptions{
		downloadbase: "releases",
		downloadtype: "resources",
		localArchive: path.Join(LocalRoot(), "packages", "downloads"),
		host:         LOCAL,
	}
	for _, opt := range options {
		opt(d)
	}
	return
}

// DownloadOnly prevents the unarchiving of the selected packages.
// Downloads are stored in the directory given to Source() or the
// default `packages/download` directory. NoSave and DownloadOnly
// are mutually exclusive.
func DownloadOnly(download bool) PackageOptions {
	return func(d *packageOptions) {
		d.downloadonly = download
	}
}

// Destination sets the destination host for the installation. This is
// used to determine the OS and architecture required for any download.
func Destination(host *Host) PackageOptions {
	return func(po *packageOptions) {
		po.host = host
	}
}

// NoSave stops downloads from being saved in the archive directory.
// NoSave and DownloadOnly are mutually exclusive.
func NoSave(nosave bool) PackageOptions {
	return func(d *packageOptions) { d.nosave = nosave }
}

// LocalOnly uses only existing archives and prevents attempts to
// download releases
func LocalOnly(local bool) PackageOptions {
	return func(d *packageOptions) { d.localOnly = local }
}

// LocalArchive is the local archive location or the specific release
// file. It can be a directory, in which case that directory is used for
// the appropriate archive file(s)
func LocalArchive(path string) PackageOptions {
	return func(d *packageOptions) { d.localArchive = path }
}

// Force ignores existing directories or files and also overrides
// protection for running instances in upgrades
func Force(force bool) PackageOptions {
	return func(d *packageOptions) { d.force = force }
}

// OverrideVersion forces a specific version to be used and fails if not
// available
func OverrideVersion(version string) PackageOptions {
	return func(d *packageOptions) { d.override = version }
}

// Restart sets the instances to be restarted around the update
func Restart(instance ...Instance) PackageOptions {
	return func(d *packageOptions) {
		d.restart = append(d.restart, instance...)
	}
}

// DoUpdate sets the option to also do an update after an install
func DoUpdate(update bool) PackageOptions {
	return func(d *packageOptions) { d.doupdate = update }
}

// StartFunc sets the start function to call for each instance given in
// Restart(). It is required to avoid import loops.
func StartFunc(fn func(Instance, ...any) error) PackageOptions {
	return func(d *packageOptions) {
		d.start = fn
	}
}

// StopFunc sets the start function to call for each instance given in
// Restart(). It is required to avoid import loops.
func StopFunc(fn func(Instance, bool, bool) error) PackageOptions {
	return func(d *packageOptions) {
		d.stop = fn
	}
}

// Version sets the desired version number, defaults to "latest" in most
// cases. The version number is in the form `[GA]X.Y.Z` (or `RA` for
// snapshots)
func Version(version string) PackageOptions {
	return func(d *packageOptions) { d.version = version }
}

// Basename sets the package binary basename, defaults to active_prod,
// for symlinks for update.
func Basename(basename string) PackageOptions {
	return func(d *packageOptions) { d.basename = basename }
}

// SetPlatformID sets the (Linux) platform ID from the OS release info.
// Currently used to distinguish RHEL8/9 releases from others.
func SetPlatformID(platform string) PackageOptions {
	return func(d *packageOptions) { d.platformId = platform }
}

// UseRoot sets the Geneos installation home directory (aka `geneos` in
// the settings)
func UseRoot(root string) PackageOptions {
	return func(d *packageOptions) { d.geneosdir = root }
}

// Username is the remote access username for downloads
func Username(username string) PackageOptions {
	return func(d *packageOptions) { d.username = username }
}

// Password is the remote access password for downloads
func Password(password *config.Plaintext) PackageOptions {
	return func(d *packageOptions) { d.password = password }
}

// UseNexus sets the flag to use nexus.itrsgroup.com for internal
// downloads instead of the default download URL in the settings. This
// also influences the way the remote path is searched and build, not
// just the base URL.
func UseNexus() PackageOptions {
	return func(d *packageOptions) { d.downloadtype = "nexus" }
}

// UseNexusSnapshots set the flag to use Nexus Snapshots rather than
// Releases.
func UseNexusSnapshots() PackageOptions {
	return func(d *packageOptions) { d.downloadbase = "snapshots" }
}
