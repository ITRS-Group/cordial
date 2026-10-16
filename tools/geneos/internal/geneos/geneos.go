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
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DeRuina/timberjack"
	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/spf13/cobra"
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
// The only options supported are:
//
//	geneos.UseRoot(dir string) // to set the root directory of the installation
//	geneos.Force(force bool) // to force init if the directory already exists and is not empty
func Initialise(cmd *cobra.Command, h *Host, options ...PackageOption) (err error) {
	opts := evalOptions(options...)
	if opts.geneosdir == "" {
		log.Error("homedir not set")
		os.Exit(1)
		// default or error
	}

	opts.geneosdir, _ = h.Abs(opts.geneosdir)

	// dir must first not exist (or be empty) and then be creatable
	//
	// TODO: maybe check that the entire list of registered directories
	// are either directories or do not exist
	if _, err := h.Stat(opts.geneosdir); err != nil {
		if err = h.MkdirAll(opts.geneosdir, 0775); err != nil {
			log.Error("cannot create directory", slog.Any("error", err))
			os.Exit(1)
		}
	} else if !opts.force {
		// check empty
		dirs, err := h.ReadDir(opts.geneosdir)
		if err != nil {
			log.Error("cannot read directory", slog.Any("error", err))
			os.Exit(1)
		}
		for _, entry := range dirs {
			if !strings.HasPrefix(entry.Name(), ".") {
				if h != LOCAL {
					log.Debug("remote directories exist, exiting init")
					return nil
				}
				log.Error("target directory exists and is not empty", slog.String("directory", opts.geneosdir))
				os.Exit(1)
			}
		}
	}

	if h.IsLocalhost() {
		log.Debug("setting geneos directory", slog.String("executable", cordial.ExecutableName()), slog.String("directory", opts.geneosdir))
		config.Set(config.Global(), cordial.ExecutableName(), opts.geneosdir)
		if err = SaveGlobalConfig(cordial.ExecutableName()); err != nil {
			return err
		}

		// recreate LOCAL to load "geneos" and others
		LOCAL = nil
		LOCAL = NewHost(LOCALHOST)
		h = LOCAL

		// initialise the global audit logger and log the original command
		if err = InitGlobalAudit(); err != nil {
			log.Error("failed to initialize global audit", slog.Any("error", err))
		}
		var cmdLine strings.Builder
		for _, a := range os.Args {
			if strings.Contains(a, " ") {
				cmdLine.WriteString(` "`)
				cmdLine.WriteString(a)
				cmdLine.WriteString(`"`)
			} else {
				cmdLine.WriteString(" ")
				cmdLine.WriteString(a)
			}
		}
		logger.Audit().Event(cmd.Name(), slog.String("command", strings.TrimSpace(cmdLine.String())))
	}

	for _, ct := range AllComponents() {
		if err := ct.MakeDirs(h); err != nil {
			continue
		}
		if ct.Initialise != nil {
			if err := ct.Initialise(h, ct); err != nil {
				log.Error("initialising component", slog.String("component", ct.Name), slog.Any("error", err))
			}
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

// InitGlobalAudit initialises the global audit logging. It returns an
// error if the audit directory cannot be created.
func InitGlobalAudit() (err error) {
	cf := config.Global()

	// TODO: This doesn't work until the directory structure is built,
	// but do not error
	if LocalRoot() == "" {
		return nil
	}

	defaultFile := filepath.Join(LocalRoot(), "audit", "audit.log")
	if err = os.MkdirAll(filepath.Dir(defaultFile), 0750); err != nil {
		return
	}

	// initialise global audit logging after loading global config,
	// which may contain defaults etc.
	logger.NewGlobalAuditLogger(&timberjack.Logger{
		Filename:         config.Get[string](cf, cf.Join("audit", "file"), config.DefaultValue(defaultFile)),
		MaxSize:          config.Get[int](cf, cf.Join("audit", "max-size"), config.DefaultValue(config.DefaultValue(10))),
		MaxAge:           config.Get[int](cf, cf.Join("audit", "max-age"), config.DefaultValue(config.DefaultValue(30))),
		MaxBackups:       config.Get[int](cf, cf.Join("audit", "max-backups"), config.DefaultValue(config.DefaultValue(5))),
		Compression:      config.Get[string](cf, cf.Join("audit", "compression"), config.DefaultValue(config.DefaultValue("gzip"))),
		BackupTimeFormat: "20060102150405",
		RotationInterval: time.Hour * 24 * 7,
		RotateAt:         []string{"00:00"},
	})
	return nil
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
		if k == "" {
			continue
		}

		// don't set key if value is the current default

		if _, ok := v.(string); !ok {
			// skip non-string values
			continue
		}

		if v == defaultSettings[k] {
			continue
		}

		config.Set(cf, k, v)
	}

	return cf.Write(name)
}
