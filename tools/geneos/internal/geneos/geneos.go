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

// The `geneos` package provides internal features to manage a typical
// `Best Practice` installation layout and the conventions that have
// formed around that structure over many years.
package geneos

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/rs/zerolog/log"
)

var (
	ErrInvalidArgs  error = errors.New("invalid arguments")
	ErrNotSupported error = errors.New("not supported")
	ErrDisabled     error = errors.New("disabled")
	ErrIsADirectory error = errors.New("is a directory")
)

const DisableExtension = "disabled"

var RootCAFile = "rootCA"
var SigningCertFile = Execname
var ConfigFileType = "json"
var GlobalConfigDir = "/etc"
var ConfigSubdirName = Execname
var UserConfigFile = "geneos.json"

// Init initialises a Geneos environment by creating a directory
// structure and then it calls the initialisation functions for each
// component type registered.
//
// If the directory is not empty and the Force() option is not passed
// then nothing is changed
func Init(h *Host, options ...Options) (err error) {
	opts := EvalOptions(options...)
	if opts.homedir == "" {
		log.Fatal().Msg("homedir not set")
		// default or error
	}

	// dir must first not exist (or be empty) and then be creatable
	//
	// XXX maybe check that the entire list of registered directories
	// are either directories or do not exist
	if _, err := h.Stat(opts.homedir); err != nil {
		if err = h.MkdirAll(opts.homedir, 0775); err != nil {
			log.Fatal().Err(err).Msg("")
		}
	} else if !opts.force {
		// check empty
		dirs, err := h.ReadDir(opts.homedir)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		for _, entry := range dirs {
			if !strings.HasPrefix(entry.Name(), ".") {
				if h != LOCAL {
					log.Debug().Msg("remote directories exist, exiting init")
					return nil
				}
				log.Fatal().Msgf("target directory %q exists and is not empty", opts.homedir)
			}
		}
	}

	if h == LOCAL {
		config.Set(Execname, opts.homedir)
		if err = config.Save(Execname); err != nil {
			return err
		}

		// recreate LOCAL to load "geneos" and others
		LOCAL = nil
		LOCAL = NewHost(LOCALHOST)
		h = LOCAL
	}

	for _, c := range AllComponents() {
		if err := c.MakeComponentDirs(h); err != nil {
			continue
		}
		if c.Initialise != nil {
			c.Initialise(h, c)
		}
	}

	return
}

// Root return the absolute path to the local Geneos installation. If
// run on an older installation it may return the value from the legacy
// configuration item `itrshome` if `geneos` is not set.
func Root() string {
	return config.GetString(Execname, config.Default(config.GetString("itrshome")))
}

// ReadLocalConfigFile reads a local configuration file without the need
// for a host connection, primarily for bootstrapping
func ReadLocalConfigFile(file string, config interface{}) (err error) {
	jsonFile, err := os.ReadFile(file)
	if err != nil {
		return
	}

	// dec := json.NewDecoder(jsonFile)
	return json.Unmarshal(jsonFile, &config)
}
