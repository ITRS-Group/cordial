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

type Options func(*configOptions)

func evalOptions(configName string, options ...Options) (c *configOptions) {
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

// Global tells [LoadConfig] to set values in the global configuration
// structure instead of creating a new one. The global configuration is
// returned by [LoadConfig].
func Global() Options {
	return func(c *configOptions) {
		c.setglobals = true
	}
}

// UseDefaults tells [LoadConfig] whether to load defaults or not. The
// default is true.
func UseDefaults(b bool) Options {
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
func SetDefaults(defaults []byte, format string) Options {
	return func(c *configOptions) {
		c.defaults = defaults
		c.defaultsFormat = format
	}
}

// SetAppName overrides to use of the [LoadConfig] `configName` argument
// as the application name, `AppName`, which is used for sub-directories
// while `configName“ is used as the prefix for files in those
// directories.
func SetAppName(name string) Options {
	return func(c *configOptions) {
		c.appname = name
	}
}

// SetConfigFile forces [LoadConfig] to load only the configuration at
// the given path. This path must include the file extension. Defaults
// are still loaded using the normal directories unless [IgnoreDefaults]
// is also passed as an option.
func SetConfigFile(path string) Options {
	return func(c *configOptions) {
		c.configFile = path
	}
}

// AddConfigDirs adds one or more directories to search for the
// configuration and defaults files. Directories are searched in order,
// so any directories set with this option are checked before the
// built-in list. This option can be given multiple times as each call
// appends to the existing list..
func AddConfigDirs(paths ...string) Options {
	return func(c *configOptions) {
		c.configDirs = append(c.configDirs, paths...)
	}
}

// IgnoreWorkingDir tells [LoadConfig] not to search the working
// directory for configuration files. This should be used when the
// caller may be running from an unknown or untrusted location.
func IgnoreWorkingDir() Options {
	return func(c *configOptions) {
		c.workingdir = ""
	}
}

// IgnoreUserConfDir tells [LoadConfig] not to search under the user
// config directory (The user configuration directory is as per
// [os.UserConfDir] and a sub-directory of AppName)
func IgnoreUserConfDir() Options {
	return func(c *configOptions) {
		c.userconfdir = ""
	}
}

// IgnoreSystemDir tells LoadConfig() not to search in the system
// configuration directory. This only applies on UNIX-like systems and
// is normally `/etc` and a sub-directory of AppName.
func IgnoreSystemDir() Options {
	return func(c *configOptions) {
		c.systemdir = ""
	}
}

// MergeSettings change the default behaviour of [LoadConfig] which is
// to load the first configuration file found, instead loading each
// configuration file found and merges the settings together. Merging is
// done using [viper.MergeConfigMap] and should result in the last
// definition of each configuration item being used.
//
// MergeSettings applies to both default and main settings, but
// separately.
func MergeSettings() Options {
	return func(c *configOptions) {
		c.merge = true
	}
}
