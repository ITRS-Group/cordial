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

// Package geneos provides internal features to manage a typical `Best
// Practice` installation layout and the conventions that have formed
// around that structure over many years.
package geneos

import (
	"errors"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/rs/zerolog/log"
)

// Useful errors for the package to return
//
// Can also be used by other packages
var (
	ErrRootNotSet   = errors.New("root directory not set")
	ErrInvalidArgs  = errors.New("invalid arguments")
	ErrNotSupported = errors.New("not supported")
	ErrIsADirectory = errors.New("is a directory")
	ErrExists       = errors.New("exists")
	ErrNotExist     = errors.New("does not exist")
	ErrDisabled     = errors.New("instance is disabled")
	ErrProtected    = errors.New("instance is protected")
	ErrRunning      = errors.New("instance is running")
	ErrNotRunning   = errors.New("instance is not running")
)

// DisableExtension is the suffix added to instance config files to mark
// them disabled
const DisableExtension = "disabled"

// RootCAFile is the file base name for the root certificate authority
// created with the TLS commands
var RootCAFile = "rootCA"

// SigningCertFile is the file base name for the signing certificate
// created with the TLS commands
var SigningCertFile string

// ChainCertFile the is file name (including extension, as this does not
// need to be used for keys) for the consolidated chain file used to
// verify instance certificates
var ChainCertFile string

// Initialise a Geneos environment by creating a directory structure and
// then it calls the initialisation functions for each component type
// registered.
//
// If the directory is not empty and the Force() option is not passed
// then nothing is changed
func Initialise(h *Host, options ...PackageOptions) (err error) {
	opts := evalOptions(options...)
	if opts.geneosdir == "" {
		log.Fatal().Msg("homedir not set")
		// default or error
	}

	opts.geneosdir, _ = h.Abs(opts.geneosdir)

	// dir must first not exist (or be empty) and then be creatable
	//
	// XXX maybe check that the entire list of registered directories
	// are either directories or do not exist
	if _, err := h.Stat(opts.geneosdir); err != nil {
		if err = h.MkdirAll(opts.geneosdir, 0775); err != nil {
			log.Fatal().Err(err).Msg("")
		}
	} else if !opts.force {
		// check empty
		dirs, err := h.ReadDir(opts.geneosdir)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		for _, entry := range dirs {
			if !strings.HasPrefix(entry.Name(), ".") {
				if h != LOCAL {
					log.Debug().Msg("remote directories exist, exiting init")
					return nil
				}
				log.Fatal().Msgf("target directory %q exists and is not empty", opts.geneosdir)
			}
		}
	}

	if h.IsLocal() {
		config.Set(execname, opts.geneosdir)
		if err = config.Save(execname); err != nil {
			return err
		}

		// recreate LOCAL to load "geneos" and others
		LOCAL = nil
		LOCAL = NewHost(LOCALHOST)
		h = LOCAL
	}

	for _, ct := range AllComponents() {
		if err := ct.MakeDirs(h); err != nil {
			continue
		}
		if ct.Initialise != nil {
			ct.Initialise(h, ct)
		}
	}

	return
}

// Init is called from the main command initialisation
func Init(app string) {
	execname = app
	SigningCertFile = execname
	ChainCertFile = execname + "-chain.pem"
	RootComponent.Register(nil)
}

// LocalRoot return the absolute path to the local Geneos installation. If
// run on an older installation it may return the value from the legacy
// configuration item `itrshome` if `geneos` is not set.
func LocalRoot() string {
	return config.GetString(execname, config.Default(config.GetString("itrshome")))
}
