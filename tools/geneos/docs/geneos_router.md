# `geneos router`

An `router` component is an instance of the Geneos Netprobe Router.

## Configuration

### Initial Setup

The Netprobe Router is created with a default configuration file based on a template embedded in `geneos`. If you have upgraded `geneos` from an earlier release you can extract this template (and update others) using the `geneeos init templates` command. The default template is derived from the example in the `one-to-one` directory of the Netprobe Router package.

### Instance Parameters

For general instance parameters, applicable to all component types, please see the documentation for the `geneos set` command, i.e. `geneos help set`.

The parameters described below are specific to the Netprobe Router component.

* `port` (Default: `4317`)

  The default port that the Netprobe Router listens on for incoming data streams. This is passed as the environment variable `COLLECTOR_PORT` and is used in the default template. In future releases this may be supported as a list of ports.

* `router-name` (Default: Instance Name)

  The Netprobe Router's name, which can be different to the instance name. This is used in the default templates and passed as an environment variable `APP`.

* `reporter-host` (Default: `localhost`)
* `reporter-port` (Default: `4318`)

  The host and port to which the Netprobe Router connects to for outgoing connections. These are passed as environment variables `REPORTER_HOST` and `REPORTER_PORT` and are used in the default template. In real world configurations, there may be multiple outgoing connections and support for this may be added in future releases of Cordial.

* `hostname` (Default: the OS host name or `localhost`)

  The hostname that the Netprobe Router uses to report self-monitoring metrics if they are enabled in the configuration file. This is passed as an environment variable `HOSTNAME`.

* `xms` (Default: `512m`)
* `xmx` (Default: `1024m`)

  The minimum and maximum heap size for the Router process. These are passed to the JVM as `-Xms` and `-Xmx` options. If the parameters are set with the `-Xms` and `-Xmx` prefixes then the values are passed to the JVM as-is. Otherwise the values are passed with the `-Xms` and `-Xmx` prefixes added.

* `logback` (Default: `logback.xml`)

  The name of the logback configuration file to use. By default the `logback.xml` file packaged with the installed distribution is copied into the instance directory at creation time. This will not be updated when any package is updated. If you want to use a different logback configuration file then you can either edit the `logback.xml` file in the instance directory or set this parameter to point to a different logback configuration file.

* `java-options` (Default: Unset)

  Additional options to pass to the JVM when starting the Router process. These are passed as multiple arguments split on white space, similar to the general `options` parameter. The difference is that the settings for `java-options` are passed to the JVM before the `-jar` option and the Router JAR file, whereas the general `options` parameter is passed after the `-jar` option and the Router JAR file.

* `main-class` (Default: `com.itrsgroup.collection.ca.Main`)

  The main class to run in the Router JAR file. This is passed to the JVM as the argument after the `-jar` option and the Router JAR file. The default value is the main class packaged with the installed distribution. If you want to use a different main class then you can set this parameter to point to a different main class in the Router JAR file.

* `setup` (Default: `router-config.yaml`)

  The path to the Netprobe Router configuration file. When creating the instance you can either import this file (using the `--import`/`-i` option) or copy it into the instance directory after creation. The default file is based on a template embedded in `geneos` which is derived from the example in the `one-to-one` directory of the Netprobe Router package. For example:

  ```bash
  geneos add router myrouter --import router-config.yaml=./myrouter.yaml --start
  ```

* `plugins` (Default: `plugins` in package directory)

  This parameter value is used in the default template as the value for the `pluginsDirectory` property in the Netprobe Router configuration file. It is passed as the environment variable `PLUGIN_DIR`. Note the singular for of `PLUGIN` for the environment variable, which is deliberate and based on Collection Agent configurations.


## Usage

```text
geneos router
```

### Options

```text
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
