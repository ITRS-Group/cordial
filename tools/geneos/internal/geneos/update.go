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

	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"

	"github.com/rs/zerolog/log"
)

// check selected version exists first
func Update(h *host.Host, ct *Component, options ...Options) (err error) {
	opts := EvalOptions(options...)
	if ct == nil {
		for _, t := range RealComponents() {
			if err = Update(h, t, options...); err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Error().Err(err).Msg("")
			}
		}
		return nil
	}

	if opts.version == "" {
		opts.version = "latest"
	}

	originalVersion := opts.version

	// before updating a specific type on a specific host, loop
	// through related types, hosts and components. continue to
	// other items if a single update fails?
	//
	// XXX this is a common pattern, should abstract it a bit like loopCommand

	if ct.RelatedTypes != nil {
		for _, rct := range ct.RelatedTypes {
			if err = Update(h, rct, options...); err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Error().Err(err).Msg("")
			}
		}
		return nil
	}

	if h == host.ALL {
		for _, h := range host.AllHosts() {
			if err = Update(h, ct, options...); err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Error().Err(err).Msg("")
			}
		}
		return
	}

	// from here hosts and component types are specific

	log.Debug().Msgf("checking and updating %s on %s %q to %q", ct, h, opts.basename, opts.version)

	basedir := h.Filepath("packages", ct)
	basepath := filepath.Join(basedir, opts.basename)

	if opts.version == "latest" {
		opts.version = ""
	}
	// opts.version, err = LatestArchive(h, basedir, opts.version, func(d os.DirEntry) bool {
	// 	return d.IsDir()
	// })
	opts.version, err = LatestVersion(h, ct, opts.version)
	if err != nil {
		log.Debug().Err(err).Msg("latest() returned err")
	}

	if opts.version == "" {
		return fmt.Errorf("%q version of %s on %s: %w", originalVersion, ct, h, os.ErrNotExist)
	}

	// does the version directory exist?
	existing, err := h.Readlink(basepath)
	if err != nil {
		log.Debug().Msgf("cannot read link for existing version %s", basepath)
	}

	// before removing existing link, check there is something to link to
	if _, err = h.Stat(filepath.Join(basedir, opts.version)); err != nil {
		return fmt.Errorf("%q version of %s on %s: %w", opts.version, ct, h, os.ErrNotExist)
	}

	if (existing != "" && !opts.force) || existing == opts.version {
		return nil
	}

	if err = h.Remove(basepath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err = h.Symlink(opts.version, basepath); err != nil {
		return err
	}
	if h == host.LOCAL && utils.IsSuperuser() {
		uid, gid, _, err := utils.GetIDs(h.GetString("username"))
		if err == nil {
			host.LOCAL.Lchown(basepath, uid, gid)
		}
	}
	fmt.Println(ct, h.Path(basepath), "updated to", opts.version)
	return nil
}
