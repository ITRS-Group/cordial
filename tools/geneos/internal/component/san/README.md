A `san` component is an instance of a [Geneos Netprobe](https://docs.itrsgroup.com/docs/geneos/current/collection/netprobe/introduction/netprobe-overview/index.html), which is the primary agent used to collect data from monitored systems, running in a Self-Announcing mode.

Other component types that represent specific flavours of Netprobe, e.g. `fa2` for the FIX Analyser Netprobe, `minimal` for the non-Collection Agent enabled Netprobe etc., can also be configured as a `san` component. Do this when adding a `san` component by prefixing the instance name with the package type, e.g. `fa2:faSAN1` or `minimal:minSAN1`. The `pkgtype` (see below) is then set to the prefix, e.g. `fa2` or `minimal`, which are currently the only valid alternatives to `netprobe`, the default.

## Configuration

The SAN instance configuration is stored in the instance configuration file. This is a JSON file which is created when the instance is created and is updated when the `geneos set` and `geneos unset` commands are used to change parameters. The configuration file is stored in the instance directory as `san.json`. This file should not be edited directly but instead the `geneos set` and `geneos unset` commands should be used to change the configuration parameters.

In a future `cordial` release the configuration file may move to a YAML format for better readability but the JSON format would continue to be supported for backwards compatibility.

### Standard Parameters

Standard parameters always have values, using the defaults if not set in the configuration file. The value `${GENEOS_HOME}` is the directory of the Geneos installation, which is normally `/opt/itrs/geneos` but is normally set during initialisation. In the examples below any `${config:NAME}` is a reference to another configuration parameter, which is evaluated and substituted in the default value.

* `name` (Default: Instance Name)

  The name of the Netprobe instance. It should not be changed.

* `home` (Read Only: `${GENEOS_HOME}/netprobe/sans/${config:name}`)

  This parameter is read-only and is set based on the instance's directory. This allows you to move the instance directory and have the `home` parameter update accordingly. It is used as the working directory for the Netprobe process (and also for any Collection Agent the Netprobe starts).

* `pkgtype` (Default: `netprobe`)

  The package type for this instance. This is used to determine the kind of Netprobe. To select the `pkgtype` when creating a `san` instance you name the instance with a prefix like this: `fa2:faSAN1` or `minimal:minSAN1`. The `pkgtype` is then set to the prefix, e.g. `fa2` or `minimal`, which are currently the only valid alternatives to `netprobe`, which is the default.
  
  It should not be changed after the instance is created.

* `install` (Default: `${GENEOS_HOME}/packages/${config:pkgtype}`)

  The installation directory for Netprobe releases.

* `version` (Default: `active_prod`)

  The version of the Netprobe in the the `install` directory above. This is normally the name of a symbolic version (the "basename") which is maintained as a link to a real installation version directory. You can create new symbolic version or tie an instance to an exact installed version. See the `geneos package install` and `geneos package update` commands for more details.

* `binary` (Default: `netprobe.linux_64` or `fix-analyser2-netprobe.linux_64` depending on `pkgtype`)

  The Netprobe program filename. Should not be changed.

* `program` (Default: `${config:install}/${config:version}/${config:binary}`)

  The full path to the Netprobe executable. The items in the default of the form `${config:NAME}` refer other configuration parameters above.

* `setup` (Default: `${config:home}/san.setup.xml`)

  The SAN setup file. This is generated from the template file defined in `config::template` when `geneos rebuild` is run (see below). It can be edited manually but any changes will be lost when `geneos rebuild` is run again. Either set `config::rebuild` to `never` or change the template file or use the `geneos set` command to change configuration parameters that are used in the template file.

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

* `listenip` (Default: `none`)

  The IP address for the SAN process to listen on. Normally a SAN does not listen for incoming connections and so the default is `none` to stop the process from opening a listening port. It may be necessary to enable the listening port, for example to enable an API endpoint, but then the `listenip` could be set to `localhost` or a specific IP address to limit which interfaces the SAN listens on.

* `hostname` (Default: `localhost`)

  The `HOSTNAME` environment variable is set to this value when the SAN is started. It is required by the Collection Agent when self-monitoring is enabled.

* `tls::certificate` (Default: `${config:home}/gateway.pem`)
* `tls::privatekey` (Default: `${config:home}/gateway.key`)
* `tls::verify` (Default: `false`)
* `tls::ca-bundle` (Default: `${GENEOS_HOME}/tls/ca-bundle.pem`)
* `tls::minimumversion` (Default: `1.2`)

  These parameters control the use of TLS for the SAN. If `tls::certificate` and `tls::privatekey` are set then TLS is enabled and the SAN is started with the appropriate options. Note that unless the value of `listenip` is set, the SAN will not listen on any open ports and the TLS settings only apply to outbound connections.The default is to have TLS enabled with the certificate and private key files in the instance home directory. If `tls::verify` is set to `true` then the SAN will verify the remote endpoints it connects to, using the trusted roots in `tls::ca-bundle`.

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

  Environment variables set for the start-up of the SAN are stored as an array of `NAME=VALUE` pairs. They should be set and unset using `geneos set -e` and `geneos unset -e` respectively to ensure consistency.

* `config::rebuild` (Default: `always`)

  The `rebuild` parameter controls how the instance responds to the `geneos rebuild` command. See below for more details.

* `config::template` (Default: `san.setup.xml.gotmpl`)

  The `template` parameter controls which template file is used to build the SAN setup file when `geneos rebuild` is run.

* `gateways` (Default: Empty)

  The list of Gateway instances that this SAN connects to. This is stored as an array of instance names. The Gateways must be defined in the same Geneos installation and the instance names must be unique across all component types. They should be set and unset using `geneos set -g` and `geneos unset -g` respectively to ensure consistency.

* `attributes` (Default: Empty)

  Custom attributes for the instance are stored as an array of `NAME=VALUE` pairs. They should be set and unset using `geneos set -a` and `geneos unset -a` respectively to ensure consistency. These attributes are not used by `cordial` itself but can be used by external tools or scripts to store additional information about the instance.

* `types`



* `variables`


