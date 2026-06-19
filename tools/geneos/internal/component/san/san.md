A `san` component is an instance of a [Geneos Netprobe](https://docs.itrsgroup.com/docs/geneos/current/collection/netprobe/introduction/netprobe-overview/index.html), which is the agent installed to collect data from monitored systems, running in a Self-Announcing mode.

Other component types that represent specific flavours of Netprobe, e.g. `fa2` for the FIX Analyser Netprobe, `minimal` for the non-Collection Agent enabled Netprobe etc., can also be configured as a `san` component. Do this when adding a `san` component by prefixing the instance name with the package type, e.g. `fa2:faSAN1` or `minimal:minSAN1`. The `pkgtype` (see below) is then set to the prefix, e.g. `fa2` or `minimal`, which are currently the only valid alternatives to `netprobe`, the default.

## Configuration

The SAN instance configuration is stored in the instance configuration file. This is a JSON file which is created when the instance is created and is updated when the `geneos set` and `geneos unset` commands are used to change parameters. The configuration file is stored in the instance directory as `san.json`. This file should not be edited directly but instead the `geneos set` and `geneos unset` commands should be used to change the configuration parameters.

In a future `cordial` release the configuration file may move to a YAML format for better readability but the JSON format would continue to be supported for backwards compatibility.

### Instance Parameters

For general instance parameters, applicable to all component types, please see the documentation for the `geneos set` command, i.e. `geneos help set`.

The parameters described below are specific to the Gateway component.

* `pkgtype` (Default: `netprobe`)

  The package type for this instance. This is used to determine the kind of Netprobe. To select the `pkgtype` when creating a `san` instance you name the instance with a prefix like this: `fa2:faSAN1` or `minimal:minSAN1`. The `pkgtype` is then set to the prefix, e.g. `fa2` or `minimal`, which are currently the only valid alternatives to `netprobe`, which is the default.
  
  It should not be changed after the instance is created.

* `setup` (Default: `${config:home}/san.setup.xml`)

  The SAN setup file. This is generated from the template file defined in `config::template` when `geneos rebuild` is run (see below). It can be edited manually but any changes will be lost when `geneos rebuild` is run again. Either set `config::rebuild` to `never` or change the template file or use the `geneos set` command to change configuration parameters that are used in the template file.

* `calogfile` (Default: `collection_agent.log`)

  The file name of the Collection Agent log file, relative to the `home` directory or an absolute path. This is only used if the Collection Agent is enabled for the Netprobe. This parameters does not control the actual log file created by the Collection Agent, which is defined in a `logback.xml` file, but is used for the `geneos logs` command to locate and display the Collection Agent logs.

* `listenip` (Default: `none`)

  The IP address for the SAN process to listen on. Normally a SAN does not listen for incoming connections and so the default is `none` to stop the process from opening a listening port. It may be necessary to enable the listening port, for example to enable an API endpoint, but then the `listenip` could be set to `localhost` or a specific IP address to limit which interfaces the SAN listens on.

* `hostname` (Default: `localhost`)

  The `HOSTNAME` environment variable is set to this value when the SAN is started. It is required by the Collection Agent when self-monitoring is enabled.

### Configuration Parameters for `geneos rebuild`

When creating a new SAN a new setup file (`netprobe.setup.xml`) is created in the instance directory and the `setup` parameter is set to point to this file.

The following parameters control how, and if, this file is rebuilt automatically and which template to use in this case:

* `config::rebuild` (Default: `always`)

  The `rebuild` parameter controls how the instance responds to the `geneos rebuild` command. See below for more details.

* `config::template` (Default: `san.setup.xml.gotmpl`)

  The `template` parameter controls which template file is used to build the SAN setup file when `geneos rebuild` is run.

The following structured parameters are used in the default `san.setup.xml.gotmpl` template and so can be set to control the content of the generated setup file. If you change the template then these parameters may not be used.

* `gateways` (Default: Empty)

  The list of Gateway instances that this SAN will attempt to connect to. When setting these values the format is `HOSTNAME:PORT`. If the SAN is configured to support TLS then the connection to the Gateway(s) will be set to use secure mode *except* in the case when the Gateway port is `7039` which is the default insecure port.

* `attributes` (Default: Empty)

  Attributes are set using `--attribute`/`-a` with a value in the form `NAME=VALUE`. Note that the `NAME` is case-sensitive, unlike basic parameters. To remove an attribute use `geneos unset -a NAME`.

* `types` (Default: Empty)

  Types are set using `--type`/`-t` and are just the `NAME` of the type. The `NAME` is case-sensitive. To remove a type use `geneos unset -t NAME`.

* `variables` (Default: Empty)

  Geneos User Variables are set using `--variable`/`-v` and have the format `[TYPE]:NAME=VALUE`, where `TYPE` in this case is the type of content the variable stores. The supported variable `TYPEs` are: (`string`, `integer`, `double`, `boolean`, `activeTime`, `externalConfigFile` and `secret`). These `TYPE` names are case sensitive and so, for example, `String` is not a valid variable `TYPE`. Other types may be supported in the future. Variable `NAME` must be unique and setting a variable with the name of an existing one will overwrite not just the `VALUE` but also the `TYPE`.

  The `secret` type is specific to cordial and is used to store secrets in an encoded form. If you do not set a `VALUE` then you will be prompted for one, if you are running `geneos` from a terminal. If the `VALUE` is already encoded in expandable for, e.g. from `geneos aes password` then that will be used without change. Any other `VALUE` will be encoded using your user keyfile and later converted, for Gateway templates, to a Geneos style `stdAESPassword` using the Gateway keyfile. The `secret` type is not valid for other component types.

  Note that in all cases `NAME` is case-sensitive, unlike basic parameters. To remove a variable use `geneos unset -v NAME`.
