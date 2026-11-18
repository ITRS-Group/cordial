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

package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
)

// Read reads configuration values for module from internal defaults,
// external defaults and configuration files or an io.Reader. The
// directories searched and the configuration file names can be
// controlled using options. The first match is read and returned unless
// the config.MergeSettings() option is used, in which case all defaults
// are merged and then all non-defaults are merged in the order they
// were given.
//
// Examples:
//
//	config.Read("geneos", config.SetGlobal())
//
//	//go:embed somefile.json
//	var myDefaults []byte
//
//	cf, err := config.Read("geneos",
//	  config.WithDefaults(myDefaults, "json"),
//	  config.SetConfigFile(configPath),
//	)
//	if err != nil {
//	  ...
//
// Options can be passed to change the default behaviour and to pass any
// embedded defaults or an existing viper.
//
// for defaults see:
// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
//
// The order of precedence of the settings in the returned
// configuration, highest first, is: values set explicitly using [Set]
// and environment variables (if enabled with [WithEnvs]); the main
// configuration file; defaults files; the embedded defaults from
// [WithDefaults]; and last of all the settings given to [DefaultsFrom].
// None of the defaults are used if [UseDefaults] is passed false.
//
// Regardless of errors loading configurations a non-nil configuration
// object is always returned.
//
// The returned config object may be made up from multiple sources so
// there is no simple way of getting the name of the final configuration
// file used.
//
// If the LoadFrom() option is set then all file access is via the given
// remote. Defaults and the primary configuration cannot be loaded from
// different remotes. The default is "localhost".
//
// Is the SetConfigReader() option is passed to load the configuration
// from an io.Reader then this takes precedence over file discovery or
// SetConfigFile(). The configuration file format should be set with
// SetFileExtension() or it defaults as above.
func Read(module string, options ...FileOption) (cf *Config, err error) {
	opts := evalLoadOptions(module, options...)
	r := opts.remote

	if opts.setGlobals {
		// update config directory if available
		var appUserConfDir string
		if opts.userConfDir != "" {
			appUserConfDir = path.Join(opts.userConfDir, opts.appName)
		}
		cf = resetGlobal(evalFileOptions(options...), appUserConfDir)
	} else {
		cf = New(options...)
	}

	cf.rwmutex.Lock()
	cf.configType = opts.format
	cf.rwmutex.Unlock()

	// return first error after initialising the config structure.
	// Always return a config object.
	var ok bool
	if ok, err = r.IsAvailable(); !ok {
		return
	}

	cf.setFs(r.GetFs())

	// the embedded defaults are already in cf, but check them if asked
	if opts.useDefaults && len(opts.internalDefaults) > 0 && opts.internalDefaultsCheckErrors {
		internalDefaults := newScratch(options...)
		internalDefaults.setConfigType(opts.internalDefaultsFormat)
		if err = internalDefaults.readConfig(bytes.NewReader(opts.internalDefaults)); err != nil {
			return cf, fmt.Errorf("error reading internal defaults: %w", err)
		}
	}

	// concatenate config directories in order - first match wins below,
	// unless MergeSettings() option is used. The order is:
	//
	// 1. Explicit directory arguments passed using the option AddConfigDirs()
	// 2. The working directory unless the option IgnoreWorkingDir() is used
	// 3. The user configuration directory plus `AppName`, unless IgnoreUserConfDir() is used
	// 4. The system configuration directory plus `AppName`, unless IgnoreSystemDir() is used
	confDirs := slices.Clone(opts.configDirs)
	if opts.workingDir != "" {
		confDirs = append(confDirs, opts.workingDir)
	}
	if opts.userConfDir != "" {
		confDirs = append(confDirs, path.Join(opts.userConfDir, opts.appName))
	}
	if opts.systemDir != "" {
		confDirs = append(confDirs, path.Join(opts.systemDir, opts.appName))
	}

	// if we are merging, then we load in reverse order to ensure lower
	// priorities are overwritten
	if opts.merge {
		slices.Reverse(confDirs)
	}

	// search directories for defaults unless UseDefault(false) is used
	// as an option to Load(). Do this even if the config file itself is
	// set using option SetConfigFile()
	if opts.useDefaults {
		defaults := newScratch(options...)
		defaults.setFs(r.GetFs())

		if opts.merge {
			for _, dir := range confDirs {
				d := newScratch(options...)
				d.setFs(r.GetFs())
				d.setConfigFile(path.Join(dir, module+".defaults."+opts.format))
				if err = d.readInConfig(); err != nil {
					if errors.Is(err, fs.ErrNotExist) {
						// not found is fine
						continue
					}
					return cf, fmt.Errorf("error reading defaults: %w", err)
				}
				defaults.MergeConfigMap(d.AllSettings())
			}
		} else {
			for _, dir := range confDirs {
				defaults.setConfigFile(path.Join(dir, module+".defaults."+opts.format))
				if err = defaults.readInConfig(); err != nil {
					if errors.Is(err, fs.ErrNotExist) {
						continue
					}
					return cf, fmt.Errorf("error reading defaults: %w", err)
				}
				break
			}
		}

		// when we get here we have either loaded the first default
		// file or not found one.
		err = nil

		// merge all defaults into main config over the internal ones
		cf.MergeConfigMap(defaults.AllSettings())
	}

	switch {
	case opts.reader != nil:
		// fixed configuration source, skip directory search
		ncf := newScratch(options...)
		ncf.setFs(r.GetFs())
		ncf.setConfigType(opts.format)

		if err = ncf.readConfig(opts.reader); err != nil {
			return cf, fmt.Errorf("error reading config: %w", err)
		}

		// merge into main config
		cf.MergeConfigMap(ncf.AllSettings())

		// there is no file to watch
		return cf, nil

	case opts.configFile != "":
		// fixed configuration file, skip directory search
		ncf := newScratch(options...)
		ncf.setFs(r.GetFs())
		ncf.setConfigFile(opts.configFile)

		if opts.format != "" {
			ncf.setConfigType(opts.format)
		}
		if err = ncf.readInConfig(); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				if opts.mustExist {
					return
				}
			} else {
				return cf, fmt.Errorf("error reading config (%s): %w", opts.configFile, err)
			}
		}

		// set the config file we found and loaded, so WatchConfig works
		cf.setConfigFile(opts.configFile)

		// merge into main config
		cf.MergeConfigMap(ncf.AllSettings())

	case opts.merge:
		// load configuration files from given directories, in order
		found := 0
		for _, dir := range confDirs {
			d := newScratch(options...)
			d.setFs(r.GetFs())
			d.setConfigFile(path.Join(dir, module+"."+opts.format))
			if err = d.readInConfig(); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					// not found is fine, we are merging
					continue
				}
				return cf, fmt.Errorf("error reading config (%s): %w", d.configFileUsed(), err)
			}
			found++
			// set the config file we found and loaded, so WatchConfig works
			cf.setConfigFile(path.Join(dir, module+"."+opts.format))

			// merge, continue on failure
			cf.MergeConfigMap(d.AllSettings())
		}
		// return an error if no files read and MustExist() set
		if found == 0 && opts.mustExist {
			return cf, fs.ErrNotExist
		}
		err = nil

	case len(confDirs) > 0:
		ncf := newScratch(options...)
		ncf.setFs(r.GetFs())
		for _, dir := range confDirs {
			ncf.setConfigFile(path.Join(dir, module+"."+opts.format))
			if err = ncf.readInConfig(); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					continue
				}
				return cf, fmt.Errorf("error reading config (%s): %w", path.Join(dir, module+"."+opts.format), err)
			}

			// set the config file we found and loaded, so WatchConfig works
			cf.setConfigFile(path.Join(dir, module+"."+opts.format))

			// merge into main config
			cf.MergeConfigMap(ncf.AllSettings())

			// first found wins
			break
		}

		// when we get here we have either loaded the first file or not
		// found one. if err then no file was found, so check
		// opts.mustexist or just give up
		if err != nil && opts.mustExist {
			return
		}
		// otherwise return no error
		err = nil
	}

	if opts.watchConfig {
		if opts.notifyOnChange != nil {
			cf.onConfigChange(opts.notifyOnChange)
		}
		cf.watchConfig()
	}

	return cf, nil
}

// Path returns the full path to the that would be opened by Load given
// the same options.
//
// If the config.MustExist() option is used then configuration files are
// tested for readability and the first matching path, base on other
// options, is returned.
//
// If no file is found but internal default have been defined then the
// string "internal defaults" is returned.
//
// If no internal defaults are defined then the string "none" is
// returned (unless config.MustExist() is used, in which case an empty
// string is returned).
func Path(module string, options ...FileOption) string {
	opts := evalLoadOptions(module, options...)
	r := opts.remote

	if opts.configFile != "" {
		if !opts.mustExist {
			return opts.configFile
		}

		if f, err := r.Open(opts.configFile); err == nil {
			f.Close()
			return opts.configFile
		}
	}

	confDirs := slices.Clone(opts.configDirs)
	if opts.workingDir != "" {
		confDirs = append(confDirs, opts.workingDir)
	}
	if opts.userConfDir != "" {
		confDirs = append(confDirs, path.Join(opts.userConfDir, opts.appName))
	}
	if opts.systemDir != "" {
		confDirs = append(confDirs, path.Join(opts.systemDir, opts.appName))
	}

	if opts.merge {
		slices.Reverse(confDirs)
	}

	filename := module
	if opts.format != "" {
		filename = fmt.Sprintf("%s.%s", filename, opts.format)
	}

	if len(confDirs) > 0 {
		for _, dir := range confDirs {
			p := path.Join(dir, filename)
			if f, err := r.Open(p); err == nil {
				f.Close()
				return p
			}
		}
		if !opts.mustExist {
			return filepath.Join(confDirs[0], filename)
		}
	}

	if opts.internalDefaults != nil {
		return "internal defaults"
	}

	// passing MustExist to Path() means we want to know if a file would
	// be found, but we don't want to return "internal defaults" or
	// "none" if not, we just want to know if a file would be found or
	// not, so return an empty string if not found and MustExist is set.
	// Otherwise return "none" if not found and no internal defaults, or
	// "internal defaults" if not found but internal defaults are
	// defined.
	if opts.mustExist {
		return ""
	}

	return "none"
}
