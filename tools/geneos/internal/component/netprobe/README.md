A `netprobe` component is an instance of a [Geneos Netprobe](https://docs.itrsgroup.com/docs/geneos/current/collection/netprobe/introduction/netprobe-overview/index.html), which is the primary agent used to collect data from monitored systems.

While there are other component types that represent specific flavours of Netprobe, e.g. `fa2` for the FIX Analyser Netprobe, `minimal` for the non-Collection Agent enabled Netprobe etc., all of these kinds can also be configured as a normal `netprobe` component, to make overall management easier. On the other hand, the `san` and `floating` component types, while Netprobes, function differently and have different configuration parameters, so they must be configured using their own types.

## Configuration

The Netprobe instance configuration is stored in the instance configuration file. This is a JSON file which is created when the instance is created and is updated when the `geneos set` and `geneos unset` commands are used to change parameters. The configuration file is stored in the instance directory as `netprobe.json`. This file should not be edited directly but instead the `geneos set` and `geneos unset` commands should be used to change the configuration parameters.

In a future `cordial` release the configuration file may move to a YAML format for better readability but the JSON format would continue to be supported for backwards compatibility.

### Standard Parameters

Standard parameters always have values, using the defaults if not set in the configuration file. The value `${GENEOS_HOME}` is the directory of the Geneos installation, which is normally `/opt/itrs/geneos` but is normally set during initialisation. In the examples below any `${config:NAME}` is a reference to another configuration parameter, which is evaluated and substituted in the default value.

* `name` (Default: Instance Name)

  The name of the Netprobe instance. It should not be changed.

* `home` (Read Only: `${GENEOS_HOME}/netprobe/netprobes/${config:name}`)

  This parameter is read-only and is set based on the instance's directory. This allows you to move the instance directory and have the `home` parameter update accordingly. It is used as the working directory for the Netprobe process (and also for any Collection Agent the Netprobe starts).

* `pkgtype` (Default: `netprobe`)

  The package type for this instance. This is used to determine the kind of Netprobe. To select the `pkgtype` when creating a `netprobe` instance you name the instance with a prefix like this: `fa2:faProbe1` or `minimal:minProbe1`. The `pkgtype` is then set to the prefix, e.g. `fa2` or `minimal`, which are currently the only valid alternatives to `netprobe`. If there is no prefix in the instance name then the `pkgtype` is set to `netprobe`.
  
  It should not be changed after the instance is created.

* `install` (Default: `${GENEOS_HOME}/packages/${config:pkgtype}`)

  The installation directory for Netprobe releases.

* `version` (Default: `active_prod`)

  The version of the Netprobe in the the `install` directory above. This is normally the name of a symbolic version (the "basename") which is maintained as a link to a real installation version directory. You can create new symbolic version or tie an instance to an exact installed version. See the `geneos package install` and `geneos package update` commands for more details.

* `binary` (Default: `netprobe.linux_64` or `fix-analyser2-netprobe.linux_64` depending on `pkgtype`)

  The Netprobe program filename. Should not be changed.

* `program` (Default: `${config:install}/${config:version}/${config:binary}`)

  The full path to the Netprobe executable. The items in the default of the form `${config:NAME}` refer other configuration parameters above.

* `libpaths` (Default: `${config:install}/${config:version}/lib64:/usr/lib64`)

  This parameter is combined with any `LD_LIBRARY_PATH` environment variable to create the `LD_LIBRARY_PATH` used when starting the Netprobe. The default is the `lib64` directory of the Netprobe installation version and the standard system library directory.

* `options` (Default: Empty)

  A space separated set of additional options to append to the command line of the Netprobe. For example, when you create a "demo" environment using `geneos init demo` the Netprobe gets a `option` of `-demo`. The contents are split on space before being passed as individual arguments; this means that it is not possible to use arguments containing spaces, such as a file path.

  To pass extra parameters to the Netprobe just once please see the `--extra`/`-x` option of the `geneos start`, `geneos restart` and `geneos deploy` commands.

* `logfile` (Default: `netprobe.log`)

  The file name of the Netprobe log file, relative to the `home` directory or an absolute path.

* `calogfile` (Default: `collection_agent.log`)

  The file name of the Collection Agent log file, relative to the `home` directory or an absolute path. This is only used if the Collection Agent is enabled for the Netprobe.

* `logdir` (Default: Unset)

  If set, it is used as the directory for the log file above. If not set (the default) then the `home` directory of the instance is used.

* `port` (Default: First available from `7036,7100-`)

  The default port to listen on. The actual default is selected from the first available port in the range defined in `netprobe::ports` in the program settings. If you have multiple Netprobes running on the same server then the `geneos add` and `geneos deploy` commands, amongst others, will automatically select the next available port in the range.

  The port range is defined in the top-level configuration as `netprobe::ports` and defaults to `7036,7100-`. You can change this using `geneos config set netprobe::ports="..."`. See the `geneos config` command for more details.

* `listenip` (Default: Not set)

  The IP address to listen on. If not set then the Netprobe listens on all available interfaces.

* `hostname` (Default: `localhost`)

  The `HOSTNAME` environment variable is set to this value when the Netprobe is started. It is required by the Collection Agent when self-monitoring is enabled.

* `tls::certificate` (Default: `${config:home}/gateway.pem`)
* `tls::privatekey` (Default: `${config:home}/gateway.key`)
* `tls::verify` (Default: `false`)
* `tls::ca-bundle` (Default: `${GENEOS_HOME}/tls/ca-bundle.pem`)
* `tls::minimumversion` (Default: `1.2`)

  These parameters control the use of TLS for the Netprobe. If `tls::certificate` and `tls::privatekey` are set then TLS is enabled and the Netprobe is started with the appropriate options. The default is to have TLS enabled with the certificate and private key files in the instance home directory. If `tls::verify` is set to `true` then the Netprobe will verify the remote endpoints it connects to, using the trusted roots in `tls::ca-bundle`.

  If `verify` is set to `true` but the `tls::ca-bundle` file does not exist then the verification chain is set to an appropriate system default, which is seleected from a list of defaults for typical Linux systems.

  Deprecated parameters for TLS are also supported for backwards compatibility but should not be used in new configurations. If you are upgrading from an older version of `cordial` there is a `geneos tls migrate` command to help you. These deprecated parameters are:

  * `certificate`
  * `privatekey`
  * `certchain`
  * `use-chain`

* `autostart` (Default: `true`)

  Gateway instances are set to be started with the default `geneos start` command. Setting `autostart` to false is different to using `geneos disable` to stop an instance from running. This can be used for instances that only need to be run occasionally or manually, for example a load monitoring Gateway instance. To start a Gateway that has `autostart` set to false you must give both the type and the name to the `geneos start` command, for example `geneos start gateway example2`.

* `protected` (Default: `false`)

  If `true` then the instance is protected from being changed or deleted by the `geneos start`, `geneos stop`, `geneos restart` or `geneos delete` and similar commands. This is useful for critical instances that should not be accidentally modified or removed. When an instance is protected, any attempt to change or delete it using the above commands will result in an error message unless the command is run with the `--force` option.

  This is different to using `geneos disable` to stop an instance from running. This can be used for instances that should not be changed or deleted, for example a production Gateway instance.

* `env` (Default: Empty)

  Environment variables set for the start-up of the Gateway are stored as an array of `NAME=VALUE` pairs. They should be set and unset using `geneos set -e` and `geneos unset -e` respectively to ensure consistency.
