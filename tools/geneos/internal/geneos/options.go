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
	"github.com/itrs-group/cordial/pkg/config"
)

// geneosOptions defines the internal options for various operations in
// the geneos package
type geneosOptions struct {
	basename     string
	doupdate     bool
	downloadbase string
	downloadtype string
	force        bool
	geneosdir    string
	local        bool
	nosave       bool
	override     string
	password     *config.Plaintext
	platformId   string
	restart      bool
	source       string
	username     string
	version      string
}

type Options func(*geneosOptions)

func EvalOptions(options ...Options) (d *geneosOptions) {
	// defaults
	d = &geneosOptions{
		downloadbase: "releases",
		downloadtype: "resources",
	}
	for _, opt := range options {
		opt(d)
	}
	return
}

// NoSave prevents downloads from being saved in the archive directory
func NoSave(n bool) Options {
	return func(d *geneosOptions) { d.nosave = n }
}

// LocalOnly stops downloads from external locations
func LocalOnly(l bool) Options {
	return func(d *geneosOptions) { d.local = l }
}

// Force ignores existing directories or files
func Force(o bool) Options {
	return func(d *geneosOptions) { d.force = o }
}

// OverrideVersion forces a specific version to be used and failure if not available
func OverrideVersion(s string) Options {
	return func(d *geneosOptions) { d.override = s }
}

// Restart sets the instances to be restarted around the update
func Restart(r bool) Options {
	return func(d *geneosOptions) { d.restart = r }
}

// Restart returns the value of the Restart option. This is a helper to
// allow checking outside the cmd package.
// XXX Currently doesn't work.
func (d *geneosOptions) Restart() bool {
	return d.restart
}

// Version sets the desired version number, defaults to "latest" in most
// cases. The version number is in the form `[GA]X.Y.Z` (or `RA` for
// snapshots)
func Version(v string) Options {
	return func(d *geneosOptions) { d.version = v }
}

// Basename sets the package binary basename, defaults to active_prod,
// for symlinks for update.
func Basename(b string) Options {
	return func(d *geneosOptions) { d.basename = b }
}

// UseRoot sets the Geneos installation home directory (aka `geneos` in
// the settings)
func UseRoot(h string) Options {
	return func(d *geneosOptions) { d.geneosdir = h }
}

// Username is the remote access username for downloads
func Username(u string) Options {
	return func(d *geneosOptions) { d.username = u }
}

// Password is the remote access password for downloads
func Password(p *config.Plaintext) Options {
	return func(d *geneosOptions) { d.password = p }
}

// PlatformID sets the (Linux) platform ID from the OS release info.
// Currently used to distinguish RHEL8 installs from others.
func PlatformID(id string) Options {
	return func(d *geneosOptions) { d.platformId = id }
}

// UseNexus sets the flag to use nexus.itrsgroup.com for internal
// downloads instead of the default download URL in the settings. This
// also influences the way the remote path is searched and build, not
// just the base URL.
func UseNexus() Options {
	return func(d *geneosOptions) { d.downloadtype = "nexus" }
}

// UseSnapshots set the flag to use Nexus Snapshots rather than
// Releases.
func UseSnapshots() Options {
	return func(d *geneosOptions) { d.downloadbase = "snapshots" }
}

// Source is the source of the installation and overrides all other
// settings include Local and download URLs. It can be a directory, in
// which case that directory is searched for the appropriate archive
// file(s)
func Source(f string) Options {
	return func(d *geneosOptions) { d.source = f }
}

// DoUpdate sets the option to also do an update after an install
func DoUpdate(r bool) Options {
	return func(d *geneosOptions) { d.doupdate = r }
}
