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

package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type configOptions struct {
	defaults       []byte
	defaultsFormat string
	configFile     string
	configDirs     []string
	appname        string
	workingdir     string
	userconfdir    string
	systemdir      string
	setglobals     bool
	usedefaults    bool
	merge          bool
}

// LoadOptions can be passed to the LoadConfig function to
// influence it's behaviour.
type LoadOptions func(*configOptions)

func evalOptions(configName string, options ...LoadOptions) (c *configOptions) {
	// init
	c = &configOptions{}
	c.configDirs = []string{}
	c.workingdir = "."
	c.systemdir = "/etc" // UNIX/Linux only!
	c.userconfdir, _ = UserConfigDir()
	c.usedefaults = true

	for _, opt := range options {
		opt(c)
	}

	// defaults
	if c.defaultsFormat == "" {
		c.defaultsFormat = "yaml"
	}

	if c.appname == "" {
		c.appname = configName
	}

	return
}

// SetGlobal tells [LoadConfig] to set values in the global
// configuration structure instead of creating a new one. The global
// configuration is then returned by [LoadConfig].
func SetGlobal() LoadOptions {
	return func(c *configOptions) {
		c.setglobals = true
	}
}

// UseDefaults tells [LoadConfig] whether to load defaults or not. The
// default is true. Defaults are loaded from a file with the same name
// as the main on but with an extra `.defaults` suffix before the
// extension, i.e. for `config.yaml` the defaults file would be
// `config.defaults.yaml` but it is searched in all the directories and
// may be located elsewhere to the main configuration.
func UseDefaults(b bool) LoadOptions {
	return func(c *configOptions) {
		c.usedefaults = b
	}
}

// SetDefaults takes a []byte slice and a format type to set
// configuration defaults. This can be used in conjunction with `embed`
// to set embedded default configuration values so that a program can
// function without a configuration file, e.g.
//
//	//go:embed "defaults.yaml"
//	var defaults []byte
//	...
//	c, err := config.LoadConfig("appname", config.SetDefaults(defaults, "yaml"))
func SetDefaults(defaults []byte, format string) LoadOptions {
	return func(c *configOptions) {
		c.defaults = defaults
		c.defaultsFormat = format
	}
}

// SetAppName overrides to use of the [LoadConfig] `configName` argument
// as the application name, `AppName`, which is used for sub-directories
// while `configName“ is used as the prefix for files in those
// directories.
//
// For example, if LoadConfig is called like this:
//
//	LoadConfig("myprogram", config.SetAppName("basename"))
//
// Then one valid location of a configuration file would be:
//
//	${HOME}/.config/basename/myprogram.yaml
func SetAppName(name string) LoadOptions {
	return func(c *configOptions) {
		c.appname = name
	}
}

// SetConfigFile forces [LoadConfig] to load only the configuration at
// the given path. This path must include the file extension. Defaults
// are still loaded from all the normal directories unless
// [IgnoreDefaults] is also passed as an option.
func SetConfigFile(path string) LoadOptions {
	return func(c *configOptions) {
		c.configFile = path
	}
}

// AddConfigDirs adds paths as directories to search for the
// configuration and defaults files. Directories are searched in the
// order given, and any directories added with this option are checked
// before any built-in list. This option can be given multiple times and
// each call appends to the existing list.
func AddConfigDirs(paths ...string) LoadOptions {
	return func(c *configOptions) {
		c.configDirs = append(c.configDirs, paths...)
	}
}

// IgnoreWorkingDir tells [LoadConfig] not to search the working
// directory of the process for configuration files. This should be used
// when the caller may be running from an unknown or untrusted location.
func IgnoreWorkingDir() LoadOptions {
	return func(c *configOptions) {
		c.workingdir = ""
	}
}

// IgnoreUserConfDir tells [LoadConfig] not to search under the user
// config directory. The user configuration directory is as per
// [os.UserConfDir]
func IgnoreUserConfDir() LoadOptions {
	return func(c *configOptions) {
		c.userconfdir = ""
	}
}

// IgnoreSystemDir tells LoadConfig() not to search in the system
// configuration directory. This only applies on UNIX-like systems and
// is normally `/etc` and a sub-directory of AppName.
func IgnoreSystemDir() LoadOptions {
	return func(c *configOptions) {
		c.systemdir = ""
	}
}

// MergeSettings change the default behaviour of [LoadConfig] which is
// to load the first configuration file found, instead loading each
// configuration file found and merging the settings together. Merging
// is done using [viper.MergeConfigMap] and should result in the last
// definition of each configuration item being used.
//
// MergeSettings applies to both default and main settings, but
// separately, i.e. all defaults are first merged and applied then the
// main configuration files are merged and loaded.
func MergeSettings() LoadOptions {
	return func(c *configOptions) {
		c.merge = true
	}
}

// Load reads configuration values from internal defaults, external
// defaults and configuration files. The directories searched and the
// configuration file names can be controlled using options. The first
// match is loaded unless the config.MergeSettings() option is used,
// in which case all defaults are merged and then all non-defaults are
// merged in the order they were given.
//
// Examples:
//
//	config.Load("geneos", config.SetGlobal())
//
//	//go:embed somefile.json
//	var myDefaults []byte
//	Load("geneos", config.SetDefaults(myDefaults, "json"), config.SetConfigFile(configPath))
//
// Options can be passed to change the default behaviour and to pass any
// embedded defaults or an existing viper.
//
// for defaults see:
// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
//
// TBD: windows equiv of above
func Load(configName string, options ...LoadOptions) (c *Config, err error) {
	opts := evalOptions(configName, options...)

	if opts.setglobals {
		c = global
	} else {
		c = New()
	}

	defaults := viper.New()
	internalDefaults := viper.New()

	if opts.usedefaults && len(opts.defaults) > 0 {
		buf := bytes.NewBuffer(opts.defaults)
		internalDefaults.SetConfigType(opts.defaultsFormat)
		// ignore errors ?
		internalDefaults.ReadConfig(buf)

		// now set any internal default values as real defaults, cannot use Merge here
		for k, v := range internalDefaults.AllSettings() {
			defaults.SetDefault(k, v)
		}
	}

	// concatenate config directories in order - first match wins below,
	// unless MergeSettings() option is used. The order is:
	//
	// 1. Explicit directory arguments passed using the option AddConfigDirs()
	// 2. The working directory unless the option IgnoreWorkingDir() is used
	// 3. The user configuration directory plus `AppName`, unless IgnoreUserConfDir() is used
	// 4. The system configuration directory plus `AppName`, unless IgnoreSystemDir() is used
	confDirs := opts.configDirs
	if opts.workingdir != "" {
		confDirs = append(confDirs, opts.workingdir)
	}
	if opts.userconfdir != "" {
		confDirs = append(confDirs, filepath.Join(opts.userconfdir, opts.appname))
	}
	if opts.systemdir != "" {
		confDirs = append(confDirs, filepath.Join(opts.systemdir, opts.appname))
	}

	// if we are merging, then we load in reverse order to ensure lower
	// priorities are overwritten
	if opts.merge {
		for i := len(confDirs)/2 - 1; i >= 0; i-- {
			opp := len(confDirs) - 1 - i
			confDirs[i], confDirs[opp] = confDirs[opp], confDirs[i]
		}
	}
	log.Debug().Msgf("confDirs: %v", confDirs)

	// search directories for defaults unless UseDefault(false) is
	// used as an option to LoadConfig(). we do this even if the
	// config file itself is set using option SetConfigFile()
	if opts.usedefaults {
		if opts.merge {
			for _, dir := range confDirs {
				d := viper.New()
				d.AddConfigPath(dir)
				d.SetConfigName(configName + ".defaults")
				d.ReadInConfig()
				if err = d.ReadInConfig(); err != nil {
					if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, fs.ErrNotExist) {
						// not found is fine
						continue
					} else {
						return c, fmt.Errorf("error reading defaults: %w", err)
					}
				}
				for k, v := range d.AllSettings() {
					defaults.SetDefault(k, v)
				}
			}
		} else if len(confDirs) > 0 {
			for _, dir := range confDirs {
				defaults.AddConfigPath(dir)
			}

			defaults.SetConfigName(configName + ".defaults")
			defaults.ReadInConfig()
			if err = defaults.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, fs.ErrNotExist) {
					// not found is fine
				} else {
					return c, fmt.Errorf("error reading defaults: %w", err)
				}
			}
		}

		// set defaults in real config based on collected defaults,
		// following viper behaviour if the same default is set multiple
		// times.
		for k, v := range defaults.AllSettings() {
			c.Viper.SetDefault(k, v)
		}
	}

	// fixed configuration file, skip directory search
	if opts.configFile != "" {
		c.Viper.SetConfigFile(opts.configFile)
		if err = c.Viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, fs.ErrNotExist) {
				// not found is fine
			} else {
				return c, fmt.Errorf("error reading config: %w", err)
			}
		}
		return c, nil
	}

	// load configuration files from given directories, in order

	if opts.merge {
		for _, dir := range confDirs {
			d := viper.New()
			d.AddConfigPath(dir)
			d.SetConfigName(configName)
			if err = d.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, fs.ErrNotExist) {
					// not found is fine
					continue
				} else {
					return c, fmt.Errorf("error reading config: %w", err)
				}
			}
			if err = c.Viper.MergeConfigMap(d.AllSettings()); err != nil {
				log.Debug().Err(err).Msgf("merge of %s/%s failed, continuing.", dir, configName)
			}
		}
		return c, nil
	}

	if len(confDirs) > 0 {
		for _, dir := range confDirs {
			c.Viper.AddConfigPath(dir)
		}
		c.Viper.SetConfigName(configName)
		if err = c.Viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, fs.ErrNotExist) {
				// not found is fine
			} else {
				return c, fmt.Errorf("error reading config: %w", err)
			}
		}
	}

	return c, nil
}
