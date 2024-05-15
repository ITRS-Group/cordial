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
	"path"
	"slices"

	"github.com/spf13/viper"
)

// Load reads configuration values from internal defaults, external
// defaults and configuration files. The directories searched and the
// configuration file names can be controlled using options. The first
// match is loaded unless the config.MergeSettings() option is used, in
// which case all defaults are merged and then all non-defaults are
// merged in the order they were given.
//
// Examples:
//
//		config.Load("geneos", config.SetGlobal())
//
//		//go:embed somefile.json
//		var myDefaults []byte
//
//		cf, err := config.Load("geneos",
//	      config.WithDefaults(myDefaults, "json"),
//	      config.SetConfigFile(configPath),
//	    )
//		if err != nil {
//		  ...
//
// Options can be passed to change the default behaviour and to pass any
// embedded defaults or an existing viper.
//
// for defaults see:
// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
//
// Regardless of errors loading configurations a configuration object is
// always returned.
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
//
// TBD: windows equiv of above
func Load(name string, options ...FileOptions) (cf *Config, err error) {
	opts := evalLoadOptions(name, options...)
	r := opts.remote

	if opts.setglobals {
		ResetConfig(options...)
		cf = global
		// update config directory
		cf.appUserConfDir = path.Join(opts.userconfdir, opts.appname)
	} else {
		cf = New(options...)
	}

	// return first error after initialising the config structure.
	// Always return a config object.
	var ok bool
	if ok, err = r.IsAvailable(); !ok {
		return
	}

	cf.Viper.SetFs(r.GetFs())

	defaults := New(options...)
	internalDefaults := New(options...)

	if opts.usedefaults && len(opts.internalDefaults) > 0 {
		buf := bytes.NewBuffer(opts.internalDefaults)
		internalDefaults.Viper.SetConfigType(opts.internalDefaultsFormat)
		if err = internalDefaults.Viper.ReadConfig(buf); err != nil && opts.internalDefaultsCheckErrors {
			return
		}
		defaults.MergeConfigMap(internalDefaults.AllSettings())
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
		confDirs = append(confDirs, path.Join(opts.userconfdir, opts.appname))
	}
	if opts.systemdir != "" {
		confDirs = append(confDirs, path.Join(opts.systemdir, opts.appname))
	}

	// if we are merging, then we load in reverse order to ensure lower
	// priorities are overwritten
	if opts.merge {
		slices.Reverse(confDirs)
	}

	// search directories for defaults unless UseDefault(false) is used
	// as an option to Load(). Do this even if the config file itself is
	// set using option SetConfigFile()
	if opts.usedefaults {
		if opts.merge {
			for _, dir := range confDirs {
				d := New(options...)
				d.Viper.SetFs(r.GetFs())
				d.Viper.SetConfigFile(path.Join(dir, name+".defaults."+opts.extension))
				if err = d.Viper.ReadInConfig(); err != nil {
					if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, fs.ErrNotExist) {
						// not found is fine
						continue
					} else {
						return cf, fmt.Errorf("error reading defaults: %w", err)
					}
				}
				defaults.MergeConfigMap(d.AllSettings())
			}
		} else if len(confDirs) > 0 {
			for _, dir := range confDirs {
				defaults.Viper.SetFs(r.GetFs())
				defaults.Viper.SetConfigFile(path.Join(dir, name+".defaults."+opts.extension))
				if err = defaults.Viper.ReadInConfig(); err != nil {
					if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, fs.ErrNotExist) {
						// not found is fine
						continue
					} else {
						return cf, fmt.Errorf("error reading defaults: %w", err)
					}
				}
				break
			}
			// when we get here we have either loaded the first default
			// file or not found one. clear err
			err = nil
		}
		cf.MergeConfigMap(defaults.AllSettings())
	}

	// fixed configuration file, skip directory search
	if opts.configFileReader != nil {
		cf.Viper.SetConfigType(opts.extension)
		if err = cf.Viper.ReadConfig(opts.configFileReader); err != nil {
			return cf, fmt.Errorf("error reading config: %w", err)
		}
		return cf, nil
	} else if opts.configFile != "" {
		cf.Viper.SetConfigFile(opts.configFile)
		if err = cf.Viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, fs.ErrNotExist) {
				if opts.mustexist {
					return
				}
			} else {
				return cf, fmt.Errorf("error reading config (%s): %w", opts.configFile, err)
			}
		}
		return cf, nil
	}

	// load configuration files from given directories, in order
	if opts.merge {
		found := 0
		for _, dir := range confDirs {
			d := New(options...)
			d.Viper.SetFs(r.GetFs())
			d.Viper.SetConfigFile(path.Join(dir, name+"."+opts.extension))
			if err = d.Viper.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, fs.ErrNotExist) {
					// not found is fine, we are merging
					continue
				} else {
					return cf, fmt.Errorf("error reading config (%s): %w", d.Viper.ConfigFileUsed(), err)
				}
			}
			found++
			// merge, continue on failure
			cf.MergeConfigMap(d.AllSettings())
		}
		// return an error if no files read and MustExist() set
		if found == 0 && opts.mustexist {
			return cf, fs.ErrNotExist
		}
		return cf, nil
	}

	if len(confDirs) > 0 {
		ncf := New(options...)
		for _, dir := range confDirs {
			ncf.Viper.SetConfigFile(path.Join(dir, name+"."+opts.extension))
			if err = ncf.Viper.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, fs.ErrNotExist) {
					continue
				} else {
					return nil, fmt.Errorf("error reading config (%s): %w", path.Join(dir, name+"."+opts.extension), err)
				}
			}

			// set the config file we found and loaded, so WatchConfig works
			cf.Viper.SetConfigFile(path.Join(dir, name+"."+opts.extension))
			// merge into main config
			cf.MergeConfigMap(ncf.AllSettings())

			// first found wins
			break
		}

		// when we get here we have either loaded the first default
		// file or not found one. if err check opts.mustexist of just
		// give up
		if err != nil && opts.mustexist {
			return
		}
		// otherwise return no error
		err = nil
	}

	if opts.watchconfig {
		if opts.notifyonchange != nil {
			cf.Viper.OnConfigChange(opts.notifyonchange)
		}
		cf.Viper.WatchConfig()
	}

	return cf, nil
}

// Path returns the full path to the first regular file found
// (potentially on a remote host if config.Remote() is used) that would
// be opened by Load given the same options. If no file is found then
// a path to the expected file in the first configured directory is
// returned. This allows for a default value to be returned for new
// files. If no directories are used then the plain filename is
// returned.
func Path(name string, options ...FileOptions) string {
	opts := evalLoadOptions(name, options...)
	r := opts.remote

	if opts.configFile != "" {
		return opts.configFile
	}

	confDirs := opts.configDirs
	if opts.workingdir != "" {
		confDirs = append(confDirs, opts.workingdir)
	}
	if opts.userconfdir != "" {
		confDirs = append(confDirs, path.Join(opts.userconfdir, opts.appname))
	}
	if opts.systemdir != "" {
		confDirs = append(confDirs, path.Join(opts.systemdir, opts.appname))
	}

	if opts.merge {
		slices.Reverse(confDirs)
	}

	filename := name
	if opts.extension != "" {
		filename = fmt.Sprintf("%s.%s", filename, opts.extension)
	}
	if len(confDirs) > 0 {
		for _, dir := range confDirs {
			p := path.Join(dir, filename)
			if st, err := r.Stat(p); err == nil && st.Mode().IsRegular() {
				return p
			}
		}
		return path.Join(confDirs[0], filename)
	}

	return filename
}
