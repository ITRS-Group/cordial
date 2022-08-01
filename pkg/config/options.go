package config

type configOptions struct {
	defaults          []byte
	defaultsFormat    string
	configFile        string
	configDirs        []string
	appname           string
	ignoreworkingdir  bool
	ignoreuserconfdir bool
	ignoresystemdir   bool
	useglobal         bool
}

type Options func(*configOptions)

func evalOptions(configName string, c *configOptions, options ...Options) {
	// init
	c.configDirs = []string{}

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
}

// UseGlobal() uses the global config instead of creating a new instance.
func UseGlobal() Options {
	return func(c *configOptions) {
		c.useglobal = true
	}
}

// SetDefaults() takes a []byte slice and a format type to load embedded
// defaults
func SetDefaults(defaults []byte, format string) Options {
	return func(c *configOptions) {
		c.defaults = defaults
		c.defaultsFormat = format
	}
}

// SetAppName() overrides to mapping of the configName to the
// application name. Application name is used for the containing
// directories, while configName is used for the files in those
// directories.
func SetAppName(name string) Options {
	return func(c *configOptions) {
		c.appname = name
	}
}

// SetConfigFile() allows the caller to override the searching for a
// config file in the given directories and instead loads only the given
// file (after defaults are loaded as normal).
func SetConfigFile(path string) Options {
	return func(c *configOptions) {
		c.configFile = path
	}
}

// AddConfigDirs() adds one or more directories to search for the
// configuration and defaults files. Directories are searched in FIFO
// order, so any directories given are checked before the built-in list.
func AddConfigDirs(paths ...string) Options {
	return func(c *configOptions) {
		c.configDirs = append(c.configDirs, paths...)
	}
}

// IgnoreWorkingDir() tells LoadConfig not to search the working
// directory for configuration files. This should be used when the
// caller may be running in an unknown location.
func IgnoreWorkingDir() Options {
	return func(c *configOptions) {
		c.ignoreworkingdir = true
	}
}

// IgnoreUserConfDir() tells LoadConfig() not to search in the user
// config directory (OS defined as per Go os.UserConfDir())
func IgnoreUserConfDir() Options {
	return func(c *configOptions) {
		c.ignoreuserconfdir = true
	}
}

// IgnoreSystemDir() tells LoadConfig() not to search in the system
// configuration directory. This only applies on UNIX-like systems and
// is normally /etc/[appName]
func IgnoreSystemDir() Options {
	return func(c *configOptions) {
		c.ignoresystemdir = true
	}
}
