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

import "github.com/itrs-group/cordial/pkg/host"

type loadOptions struct {
	appname                string
	internalDefaults       []byte
	internalDefaultsFormat string
	configFile             string
	configFileFormat       string
	remote                 host.Host
	configDirs             []string
	workingdir             string
	userconfdir            string
	systemdir              string
	setglobals             bool
	usedefaults            bool
	merge                  bool
	notfounderr            bool
}

// LoadOptions can be passed to the Load function to
// influence it's behaviour.
type LoadOptions func(*loadOptions)

func evalLoadOptions(configName string, options ...LoadOptions) (c *loadOptions) {
	// init
	c = &loadOptions{
		configFileFormat: "json",
		remote:           host.Localhost,
		configDirs:       []string{},
		workingdir:       ".",
		systemdir:        "/etc", // UNIX/Linux only!
		usedefaults:      true,
	}
	c.userconfdir, _ = UserConfigDir()

	for _, opt := range options {
		opt(c)
	}

	// defaults
	if c.internalDefaultsFormat == "" {
		c.internalDefaultsFormat = "yaml"
	}

	if c.appname == "" {
		c.appname = configName
	}

	return
}

// SetGlobal tells [Load] to set values in the global
// configuration structure instead of creating a new one. The global
// configuration is then returned by [Load].
func SetGlobal() LoadOptions {
	return func(c *loadOptions) {
		c.setglobals = true
	}
}

// UseDefaults tells [Load] whether to load defaults or not. The
// default is true. Defaults are loaded from a file with the same name
// as the main on but with an extra `.defaults` suffix before the
// extension, i.e. for `config.yaml` the defaults file would be
// `config.defaults.yaml` but it is searched in all the directories and
// may be located elsewhere to the main configuration.
func UseDefaults(b bool) LoadOptions {
	return func(c *loadOptions) {
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
//	c, err := config.Load("appname", config.SetDefaults(defaults, "yaml"))
func SetDefaults(defaults []byte, format string) LoadOptions {
	return func(c *loadOptions) {
		c.internalDefaults = defaults
		c.internalDefaultsFormat = format
	}
}

// MustExist makes Load() return an error if the configuration file is
// not found. This does not apply to defaults.
func MustExist() LoadOptions {
	return func(lo *loadOptions) {
		lo.notfounderr = true
	}
}

// SetAppName overrides to use of the [Load] `configName` argument
// as the application name, `AppName`, which is used for sub-directories
// while `configName“ is used as the prefix for files in those
// directories.
//
// For example, if Load is called like this:
//
//	Load("myprogram", config.SetAppName("basename"))
//
// Then one valid location of a configuration file would be:
//
//	${HOME}/.config/basename/myprogram.yaml
func SetAppName(name string) LoadOptions {
	return func(c *loadOptions) {
		c.appname = name
	}
}

// SetConfigFile forces [Load] to load only the configuration at the given
// path. This path must include the file extension. Defaults are still loaded
// from all the normal directories unless [IgnoreDefaults] is also passed as an
// option.
//
// If the argument is an empty string then the option is not used. This also means
// it can be called with a command line flag value which can default to an empty
// string
func SetConfigFile(path string) LoadOptions {
	return func(c *loadOptions) {
		c.configFile = path
	}
}

// SetFileFormat sets the file format for the configuration. If the type
// is not set and the configuration file loaded has an extension then
// that is used. This appliles to both defaults and main configuration
// files (but not embedded defaults). The default is "json".
func SetFileFormat(extension string) LoadOptions {
	return func(c *loadOptions) {
		c.configFileFormat = extension
	}
}

// AddConfigDirs adds paths as directories to search for the
// configuration and defaults files. Directories are searched in the
// order given, and any directories added with this option are checked
// before any built-in list. This option can be given multiple times and
// each call appends to the existing list.
func AddConfigDirs(paths ...string) LoadOptions {
	return func(c *loadOptions) {
		c.configDirs = append(c.configDirs, paths...)
	}
}

// LoadDir sets the only directory to search for the configuration
// files. It disables searching in the working directory, the user
// config directory and the system directory.
func LoadDir(dir string) LoadOptions {
	return func(lo *loadOptions) {
		lo.configDirs = []string{dir}
		lo.workingdir = ""
		lo.systemdir = ""
		lo.userconfdir = ""
	}
}

// IgnoreWorkingDir tells [Load] not to search the working
// directory of the process for configuration files. This should be used
// when the caller may be running from an unknown or untrusted location.
func IgnoreWorkingDir() LoadOptions {
	return func(c *loadOptions) {
		c.workingdir = ""
	}
}

// IgnoreUserConfDir tells [Load] not to search under the user
// config directory. The user configuration directory is as per
// [os.UserConfDir]
func IgnoreUserConfDir() LoadOptions {
	return func(c *loadOptions) {
		c.userconfdir = ""
	}
}

// IgnoreSystemDir tells Load() not to search in the system
// configuration directory. This only applies on UNIX-like systems and
// is normally `/etc` and a sub-directory of AppName.
func IgnoreSystemDir() LoadOptions {
	return func(c *loadOptions) {
		c.systemdir = ""
	}
}

// MergeSettings change the default behaviour of [Load] which is
// to load the first configuration file found, instead loading each
// configuration file found and merging the settings together. Merging
// is done using [viper.MergeConfigMap] and should result in the last
// definition of each configuration item being used.
//
// MergeSettings applies to both default and main settings, but
// separately, i.e. all defaults are first merged and applied then the
// main configuration files are merged and loaded.
func MergeSettings() LoadOptions {
	return func(c *loadOptions) {
		c.merge = true
	}
}

func LoadFrom(r host.Host) LoadOptions {
	return func(lo *loadOptions) {
		lo.remote = r
	}
}
