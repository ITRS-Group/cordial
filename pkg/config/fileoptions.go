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
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial/pkg/host"
)

var (
	defaultKeyDelimiter  = "."
	defaultFileExtension = "json"
)

type fileOptions struct {
	appname                string
	configDirs             []string
	configFile             string
	extension              string // extension without "."
	delimiter              string
	envprefix              string
	envdelimiter           string
	internalDefaults       []byte
	internalDefaultsFormat string
	merge                  bool
	mustexist              bool
	remote                 host.Host
	savedir                string // directory to save appname/appname.ext to
	setglobals             bool
	systemdir              string
	usedefaults            bool
	userconfdir            string
	workingdir             string
}

// FileOptions can be passed to the Load or Save functions to
// influence behaviour.
type FileOptions func(*fileOptions)

func evalFileOptions(options ...FileOptions) (c *fileOptions) {
	c = &fileOptions{
		delimiter:    defaultKeyDelimiter,
		envdelimiter: "_",
	}
	for _, opt := range options {
		opt(c)
	}
	return
}

func evalLoadOptions(configName string, options ...FileOptions) (c *fileOptions) {
	// init
	c = &fileOptions{
		envdelimiter: "_",
		extension:    defaultFileExtension,
		remote:       host.Localhost,
		configDirs:   []string{},
		workingdir:   ".",
		systemdir:    "/etc", // UNIX/Linux only!
		usedefaults:  true,
		delimiter:    defaultKeyDelimiter,
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

	if c.userconfdir == "" {
		c.userconfdir = filepath.Join(c.userconfdir, c.appname)
	}

	return
}

func evalSaveOptions(options ...FileOptions) (c *fileOptions) {
	c = &fileOptions{
		extension: defaultFileExtension,
		remote:    host.Localhost,
	}
	c.savedir, _ = UserConfigDir()

	for _, opt := range options {
		opt(c)
	}

	return
}

// DefaultKeyDelimiter sets the default key delimiter for all future
// calls to config.New() and config.Load(). The default is ".". You can
// use "::" if your keys are likely to contain "." such as domains, ipv4
// addresses or version numbers. Use something else if keys are likely
// to be ipv6 addresses.
func DefaultKeyDelimiter(delimiter string) {
	defaultKeyDelimiter = delimiter
}

// DefaultFileExtension sets the default file extension for all future
// calls to config.New() and config.Load(). The initial default is "json"
func DefaultFileExtension(extension string) {
	defaultFileExtension = extension
}

// UseGlobal tells [Load] to set values in the global
// configuration structure instead of creating a new one. The global
// configuration is then returned by [Load].
func UseGlobal() FileOptions {
	return func(c *fileOptions) {
		c.setglobals = true
	}
}

// UseDefaults tells [Load] whether to load defaults or not. The
// default is true. Defaults are loaded from a file with the same name
// as the main on but with an extra `.defaults` suffix before the
// extension, i.e. for `config.yaml` the defaults file would be
// `config.defaults.yaml` but it is searched in all the directories and
// may be located elsewhere to the main configuration.
func UseDefaults(b bool) FileOptions {
	return func(c *fileOptions) {
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
func SetDefaults(defaults []byte, format string) FileOptions {
	return func(c *fileOptions) {
		c.internalDefaults = defaults
		c.internalDefaultsFormat = format
	}
}

// MustExist makes Load() return an error if the configuration file is
// not found. This does not apply to default configuration files.
func MustExist() FileOptions {
	return func(lo *fileOptions) {
		lo.mustexist = true
	}
}

// SetAppName overrides to use of the Load `name` argument as the
// application name, `AppName`, which is used for sub-directories while
// `name` is used as the prefix for files in those directories.
//
// For example, if Load is called like this:
//
//	Load("myprogram", config.SetAppName("basename"))
//
// Then one valid location of a configuration file would be:
//
//	${HOME}/.config/basename/myprogram.yaml
func SetAppName(name string) FileOptions {
	return func(c *fileOptions) {
		c.appname = name
	}
}

// SetConfigFile forces Load to load only the configuration at the given
// path. This path must include the file extension. Defaults are still
// loaded from all the normal directories unless [IgnoreDefaults] is
// also passed as an option.
//
// If the argument is an empty string then the option is not used. This
// also means it can be called with a command line flag value which can
// default to an empty string
func SetConfigFile(path string) FileOptions {
	return func(c *fileOptions) {
		c.configFile = path
	}
}

// SetFileExtension sets the file extension and, by implication, the
// format for the configuration. If the type is not set and the
// configuration file loaded has an extension then that is used. This
// applies to both defaults and main configuration files (but not
// embedded defaults). The default is "json". Any leading "." is
// removed.
func SetFileExtension(extension string) FileOptions {
	return func(c *fileOptions) {
		extension = strings.TrimLeft(extension, ".")
		c.extension = extension
	}
}

// AddDirs adds paths as directories to search for the configuration and
// defaults files. Directories are searched in the order given, and any
// directories added with this option are checked before any built-in
// list. This option can be given multiple times and each call appends
// to the existing list.
func AddDirs(paths ...string) FileOptions {
	return func(c *fileOptions) {
		c.configDirs = append(c.configDirs, paths...)
	}
}

// FromDir sets the only directory to search for the configuration
// files. It disables searching in the working directory, the user
// config directory and the system directory.
func FromDir(dir string) FileOptions {
	return func(lo *fileOptions) {
		lo.configDirs = []string{dir}
		lo.workingdir = ""
		lo.systemdir = ""
		lo.userconfdir = ""
	}
}

// IgnoreWorkingDir tells [Load] not to search the working
// directory of the process for configuration files. This should be used
// when the caller may be running from an unknown or untrusted location.
func IgnoreWorkingDir() FileOptions {
	return func(c *fileOptions) {
		c.workingdir = ""
	}
}

// IgnoreUserConfDir tells [Load] not to search under the user
// config directory. The user configuration directory is as per
// [os.UserConfDir]
func IgnoreUserConfDir() FileOptions {
	return func(c *fileOptions) {
		c.userconfdir = ""
	}
}

// IgnoreSystemDir tells Load() not to search in the system
// configuration directory. This only applies on UNIX-like systems and
// is normally `/etc` and a sub-directory of AppName.
func IgnoreSystemDir() FileOptions {
	return func(c *fileOptions) {
		c.systemdir = ""
	}
}

// MergeSettings change the default behaviour of Load which is to load
// the first configuration file found, instead loading each
// configuration file found and merging the settings together. Merging
// is done using [viper.MergeConfigMap] and should result in the last
// definition of each configuration item being used.
//
// MergeSettings applies to both default and main settings, but
// separately, i.e. all defaults are first merged and applied then the
// main configuration files are merged and loaded.
func MergeSettings() FileOptions {
	return func(c *fileOptions) {
		c.merge = true
	}
}

// Host sets the source/destination for the configuration file. It
// defaults to localhost
func Host(r host.Host) FileOptions {
	return func(lo *fileOptions) {
		lo.remote = r
	}
}

// SaveDir sets the parent / top-most configuration directory to save the
// configuration. The configuration is saved in a sub-directory named
// after the application name.
func SaveDir(dir string) FileOptions {
	return func(so *fileOptions) {
		so.savedir = dir
	}
}

// KeyDelimiter sets the delimiter for keys in the configuration loaded
// with Load. This can only be changed at the time of creation of the
// configuration object so will not apply if used with SetGlobal().
func KeyDelimiter(delimiter string) FileOptions {
	return func(fo *fileOptions) {
		fo.delimiter = delimiter
	}
}

func UseEnvs(prefix string, delimiter string) FileOptions {
	return func(fo *fileOptions) {
		fo.envprefix = prefix
		fo.envdelimiter = delimiter
	}
}
