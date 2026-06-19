A `netprobe` component is an instance of a [Geneos Netprobe](https://docs.itrsgroup.com/docs/geneos/current/collection/netprobe/introduction/netprobe-overview/index.html), which is the primary agent used to collect data from monitored systems.

While there are other component types that represent specific flavours of Netprobe, e.g. `fa2` for the FIX Analyser Netprobe, `minimal` for the non-Collection Agent enabled Netprobe etc., all of these kinds can also be configured as a normal `netprobe` component, to make overall management easier. On the other hand, the `san` and `floating` component types, while Netprobes, function differently and have different configuration parameters, so they must be configured using their own types.

## Configuration

The Netprobe instance configuration is stored in the instance configuration file. This is a JSON file which is created when the instance is created and is updated when the `geneos set` and `geneos unset` commands are used to change parameters. The configuration file is stored in the instance directory as `netprobe.json`. This file should not be edited directly but instead the `geneos set` and `geneos unset` commands should be used to change the configuration parameters.

In a future `cordial` release the configuration file may move to a YAML format for better readability but the JSON format would continue to be supported for backwards compatibility.

### Instance Parameters

For general instance parameters, applicable to all component types, please see the documentation for the `geneos set` command, i.e. `geneos help set`.

The parameters described below are specific to the Netprobe component.

* `pkgtype` (Default: `netprobe`)

  The package type for this instance. This is used to determine the kind of Netprobe. To select the `pkgtype` when creating a `netprobe` instance you name the instance with a prefix like this: `fa2:faProbe1` or `minimal:minProbe1`. The `pkgtype` is then set to the prefix, e.g. `fa2` or `minimal`, which are currently the only valid alternatives to `netprobe`. If there is no prefix in the instance name then the `pkgtype` is set to `netprobe`.
  
  It should not be changed after the instance is created.

* `calogfile` (Default: `collection-agent.log`)

  The file name of the Collection Agent log file, relative to the `home` directory or an absolute path. This is only used if the Collection Agent is enabled for the Netprobe. This parameters does not control the actual log file created by the Collection Agent, which is defined in a `logback.xml` file, but is used for the `geneos logs` command to locate and display the Collection Agent logs.

* `listenip` (Default: Not set)

  The IP address to listen on. If not set then the Netprobe listens on all available interfaces.

* `hostname` (Default: `localhost`)

  The `HOSTNAME` environment variable is set to this value when the Netprobe is started. It is required by the Collection Agent when self-monitoring is enabled.
