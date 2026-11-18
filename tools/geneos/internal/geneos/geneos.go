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
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
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

var log = logger.Logger

// DisableExtension is the suffix added to instance config files to mark
// them disabled
const DisableExtension = "disabled"

// Initialise a Geneos environment on host h by creating a directory
// structure and then it calls the initialisation functions for each
// component type registered.
//
// If the directory is not empty and the Force() option is not passed
// then nothing is changed
//
// The options supported are:
//
//	geneos.UseRoot(dir string) // to set the root directory of the installation
//	geneos.Force(force bool) // to force init if the directory already exists and is not empty
//	geneos.PromptRootDir(prompt func(format string, args ...any) (string, error))
//	to provide a prompt function for the root directory
//
// The `event` parameter is used to log the initialization event in the
// global audit logger.
//
// TODO: resturcture, possibly split into smaller functions for clarity
//
//  1. create root directory, check override, not on remotes though
//  2. import certs, if provided
//  3. initialise each registered component type (optional)
//  4. log the initialization event in the global audit logger
func Initialise(event string, h *Host, options ...GeneosOption) (err error) {
	opts := evalOptions(options...)

	// 1. get root directory path
	if opts.geneosdir == "" {
		// do we have a prompt function to ask the user for the root
		// directory?
		//
		// this will not be set for non-interactive sessions and so we
		// error out
		if opts.promptRootDir != nil {
			if opts.geneosdir, err = config.UserHomeDir(); err != nil {
				log.Error("cannot find user home directory", slog.Any("error", err))
			}
			// if the directory ends in `geneos` (or whatever binary
			// name) then do not append it again
			if filepath.Base(opts.geneosdir) != cordial.ExecutableName() {
				opts.geneosdir = filepath.Join(opts.geneosdir, cordial.ExecutableName())
			}

			if dir, err := opts.promptRootDir("Geneos Directory (default %q): ", opts.geneosdir); err != nil {
				os.Exit(1)
			} else {
				log.Debug("prompted for geneos directory", slog.String("directory", dir))
				if dir != "" {
					opts.geneosdir = dir
				}
			}
		}
		if opts.geneosdir == "" {
			log.Error("homedir not set")
			os.Exit(1)
		}
	}

	// make sure the root directory is an absolute path
	opts.geneosdir, _ = h.Abs(opts.geneosdir)

	// 2. validate and create the root directory if necessary
	if _, err := h.Stat(opts.geneosdir); err != nil {
		// directory does not exist, or is not accessible, so try to
		// create and fail hard if it cannot be created. Permissions are
		// fixed as 0755 plus umask
		if err = h.MkdirAll(opts.geneosdir, 0775); err != nil {
			log.Error("cannot create directory", slog.Any("error", err))
			os.Exit(1)
		}
	} else if !opts.override {
		// if the directory exists and is not empty, handle according to
		// override option
		dirs, err := h.ReadDir(opts.geneosdir)
		if err != nil {
			log.Error("cannot read root directory", slog.Any("error", err))
			os.Exit(1)
		}
		for _, entry := range dirs {
			if !(strings.HasPrefix(entry.Name(), ".") || entry.Name() == "bin") {
				if !h.IsLocalhost() {
					log.Debug("remote directories exist, exiting init")
					os.Exit(1)
				}
				log.Error("target directory exists and is not empty (ignoring .* and bin)", slog.String("directory", opts.geneosdir))
				os.Exit(1)
			}
		}
	}

	// 3. set the geneos directory in the global configuration if on the
	// local host. on remote hosts it is part of the host config
	if h.IsLocalhost() {
		log.Debug("setting geneos directory", slog.String("executable", cordial.ExecutableName()), slog.String("directory", opts.geneosdir))
		config.Set(config.Global(), cordial.ExecutableName(), opts.geneosdir)
		if err = SaveGlobalConfig(cordial.ExecutableName(), cordial.ExecutableName()); err != nil {
			return err
		}

		// update the Geneos directory of LOCAL, which keeps the pointer
		// valid for anything that already has it, or create it
		if LOCAL == nil {
			LOCAL = NewHost(LOCALHOST)
		} else {
			config.Set(LOCAL.Config, cordial.ExecutableName(), opts.geneosdir)
		}
		h = LOCAL

		// initialise the global audit logger and log the original command event
		if err = InitGlobalAudit(); err != nil {
			log.Error("failed to initialize global audit", slog.Any("error", err))
		}
		logger.Audit().Event(event, logger.CommandLine(os.Args))
	}

	// 4. import or initialise certs
	if !opts.insecure {
		if opts.signingBundle != "" {
			err = TLSImportBundle(opts.signingBundle, "", opts.signingBundlePassword, opts.signingBundlePrompt, false)
			if err != nil {
				return err
			}
		} else {
			// never overwrite the user-wide root and signing
			// certificates from init, even when forced. Use `tls init
			// --overwrite` to replace them.
			err = TLSInit(LOCALHOST, false, certs.DefaultKeyType)
			if err != nil {
				return err
			}
		}
	}

	// 5. initialise all registered components, includeing writing out templates and include files
	if err = RootComponent.MakeDirs(h); err != nil {
		return fmt.Errorf("creating directories: %w", err)
	}
	var ct *Component
	for ct = range ct.OrList(opts.components...) {
		if err := ct.MakeDirs(h); err != nil {
			return fmt.Errorf("creating directories for component %s: %w", ct.Name, err)
		}
		if err := ct.InitialiseOnce(h, opts.override); err != nil {
			return fmt.Errorf("initialising component %s: %w", ct.Name, err)
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

// SaveGlobalConfig saves the user's global configuration file, called
// name. The settings in the existing file are always kept, with their
// original types and nested values, and the current value from the
// global configuration (config.Global) of each of the top-level keys
// given is added to or replaces it. Nothing else from the global
// configuration is saved so that default values, and values that
// only come from environment variables, are not persisted.
func SaveGlobalConfig(name string, keys ...string) error {
	oldConfDir, _ := config.UserConfigPath()
	cf, err := config.Read(name,
		config.SearchDirs(oldConfDir),
		config.MergeSources(),
		config.SkipWorkingDir(),
		config.SkipSystemDir(),
		config.UseDefaults(false),
	)
	if err != nil {
		return fmt.Errorf("reading existing configuration: %w", err)
	}

	for _, k := range keys {
		if k == "" {
			continue
		}
		config.Set(cf, k, config.Get[any](config.Global(), k, config.NoExpand()))
	}

	return cf.Write(name)
}
