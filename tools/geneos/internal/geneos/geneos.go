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

// Package geneos provides internal features to manage a typical `Best
// Practice` installation layout and the conventions that have formed
// around that structure over many years.
package geneos

import (
	"errors"
	"strings"

	"github.com/itrs-group/cordial"
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
		log.Debug().Msgf("setting %q to %q", cordial.ExecutableName(), opts.geneosdir)
		config.Set(config.Global(), cordial.ExecutableName(), opts.geneosdir)
		if err = SaveGlobalConfig(cordial.ExecutableName()); err != nil {
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
	SigningCertBasename = cordial.ExecutableName()
	DeprecatedChainCertFile = cordial.ExecutableName() + "-chain.pem"
	RootComponent.Register(nil)
}

// LocalRoot return the absolute path to the local Geneos installation. If
// run on an older installation it may return the value from the legacy
// configuration item `itrshome` if `geneos` is not set.
func LocalRoot() string {
	return config.Get[string](config.Global(), cordial.ExecutableName(),
		config.DefaultValue(config.Get[string](config.Global(), "itrshome")))
}

// SaveGlobalConfig saves the global configuration (in config.Global)
// but excludes any values that still have their defaults, by iterating
// through registered components and checking. Any empty/zero values are
// also excluded from the saved configuration.
func SaveGlobalConfig(name string) error {
	cf := config.New()
	defaultSettings := make(map[string]string)
	for _, ct := range AllComponents() {
		for k, v := range ct.GlobalSettings {
			// add main setting
			defaultSettings[k] = v
			// also add aliases
			defaultSettings[ct.ConfigAliases[k]] = v

		}
	}

	for k, v := range config.Global().AllSettings() {
		log.Debug().Msgf("checking global config key %q", k)
		if k == "" {
			panic("empty key in global config")
		}

		// don't set key if value is the current default

		if _, ok := v.(string); !ok {
			// skip non-string values
			continue
		}

		if v == defaultSettings[k] {
			log.Debug().Msgf("skipping global config key %q with default value %q", k, v)
			continue
		}

		config.Set(cf, k, v)
	}

	log.Debug().Msgf("saving global config with %#v keys", cf.AllSettings())
	return cf.Write(name) // config.SetAppName(cordial.ExecutableName()),
	// config.IgnoreEmptyValues(),

}
