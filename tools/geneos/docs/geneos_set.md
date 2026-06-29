# `geneos set`

The `set` command lets you set one or more configuration parameters for matching instances.

There are two kinds of parameters, basic `NAME=VALUE` pairs, and structured parameters which may include lists and other kinds of values, and are passed to templates for processing. Structured parameters include environment variables, include files and Gateway connection details for Self-Announcing and Floating Netprobes. The parameters that are supported depend on the component TYPE and the templates used to build the instance configuration. Parameters that are common to all component types are described in the "Common Parameters" section below, and component-specific parameters are described in the component help, e.g. `geneos help gateway` for Gateway instances.

## Basic Parameters

Note that `geneos set` will overwrite existing parameters without warning. Setting a basic parameter to an empty string will effectively remove its value. This behaviour may change in a future release and should not be relied on. To remove a parameter use the `geneos unset` command.

The `geneos set` command supports basic parameters given as `NAME=VALUE` pairs on the command line as well as options for structured or repeatable keys. Each basic parameter uses a case-insensitive `NAME` as the key in the instance configuration. You can also use `+=` or `+` to append values to an existing basic parameter (which will also create a parameter if it does not exist), e.g. `options+="-extra option"`. If the value starts with a dash it is assumed to be a new command line parameter and is appended with a space, otherwise it is appended as-is, but you are responsible for ensuring the resulting parameter is correctly formatted, e.g. paths having ":" separators.

Parameter valules can be encoded so that secrets do not appear in plain text in configuration files. Use the `--secure`/`-s` option with a parameter name and optional plaintext value. If no value is given then you will be prompted to enter the secret. The plaintext is encoded using a keyfile, either provided with the `--keyfile`/`-k` option or, for components that support it, the keyfile referenced in the instance configuration. Otherwise the user's default keyfile is used (or created) as needed. The encoded value is the same as produced by `geneos aes password`.

Environment variables can be set using the `--env`/`-e` option, which can be repeated as required, and the argument to the option should be in the format `ENV=VALUE`. In this case the `ENV` is case-sensitive. An environment variable `ENV` will be set or updated for all matching instances under the configuration key `env`. These environment variables are used to construct the start-up environment of the instance. Environments can be added to any component TYPE.

## Structured Parameters

Environment variables can be encoded so that secrets do not appear in plain text in configuration files. Use the `--secureenv`/`-E` option with a variable name and optional plaintext value. If no value is given then you will be prompted to enter the secret. The encoding process is the same as for `--secure`/`-s` above.

Include files (only for Gateways) can be set using the `--include`/`-i` option, which can be repeated. The value must me in the form `PRIORITY:PATH|URL` where priority is a number between 1 and 65534 and the PATH is either an absolute file path or relative to the Gateway instance directory. Shared include files are normally stored in a `../../shared/` directory. Alternatively, a URL can be used to refer to a read-only remote include file. As each include file must have a different priority in the Geneos Gateway configuration file, this is the value that should be used as the unique key for updating include files.

Include file parameters are passed to templates (see `geneos help gateway`) and the template may or may not add additional values to the include file section. Templates are fully configurable and may not use these values at all.

For Self-Announcing and Floating Netprobes you can add or update Gateway connection details with the `--gateway`/`-g` option. These are given in the form `HOSTNAME:PORT`. The `HOSTNAME` can also be an IP address and is not the same as the `geneos host` command labels for remote hosts being managed, but the actual network accessible hostname or IP that the Gateway is listening on. This option can also be repeated as necessary and is applied to the instance configuration through templates, see `geneos rebuild`. To remove a Gateway connection use the `geneos unset ` command.

Three more options exist for Self-Announcing Netprobes to set Attributes, Types and Variables respectively. As above these options can be repeated and will update or replace existing parameters and to remove them you should use `geneos unset`. All of these parameters depend on SAN configurations being built using template files and do not have any effect on their own. See `geneos rebuild` for more information. See the Self-Announcing Netprobe documentation for more details, e.g. `geneos help san`.

Future releases may add other specific options and also may offer a more direct way of configuring SANs and Floating Netprobes to connect to Gateway also managed by the same `geneos` program.

## Common Parameters

All components share a common set of parameters which are described in more detail below. Each component may also support type-specific parameters which are documented in the component help, e.g. `geneos help gateway`.

### Required Parameters

Required parameters always have values, using default values if not changed in the configuration file. Note that in some cases the deault is an empty value, which is different to the parameter being unset. For example, if `logdir` is unset then the default is to use the `home` directory of the instance for logs, but if `logdir` is set to an empty value then no log directory is used and the `logfile` parameter is used as an absolute path.

In the examples below, `${GENEOS_HOME}` is the directory of the Geneos installation, which is normally `/opt/itrs/geneos` but can be set during initialisation, and values like `${config:PARAMETER}` are references to another configuration parameters, which are evaluated and replaced in the resulting value.

* `name` (Default: Instance Name)

  The name of the Gateway. This is used in the default templates, under the Operating Environment created in `instance.setup.xml`. It should not be changed.

* `home` (Read Only: `${GENEOS_HOME}/gateway/gateways/${config:name}`)

  This parameter is read-only and is set based on the instance's directory. `${config:name}` is the instance name, not the Gateway name. This allows you to move the instance directory and have the `home` parameter update accordingly. It is used as the working directory for the Gateway process.

* `install` (Default: `${GENEOS_HOME}/packages/gateway`)

  The installation directory for Gateway releases

* `version` (Default: `active_prod`)

  The version of the component in the the `install` directory above. This is normally the name of a symbolic version (the "basename") which is maintained as a link to a real installation version directory. You can create new symbolic version or tie an instance to an exact installed version. See the `geneos package install` and `geneos package update` commands for more details.

* `binary` (Default: `gateway2.linux_64`)

  The component program filename. Should not be changed.

* `program` (Default: `${config:install}/${config:version}/${config:binary}`)

  The full path to the component executable. The items in the default of the form `${config:NAME}` refer other configuration parameters above.

* `libpaths` (Default: `${config:install}/${config:version}/lib64:/usr/lib64`)

  This parameter is combined with any `LD_LIBRARY_PATH` environment variable to create the `LD_LIBRARY_PATH` used when starting the component. The default is the `lib64` directory of the component installation version and the standard system library directory.

* `cpus` (Default: Empty)

  For local Linux instances, a comma separated list of CPU numbers to set the CPU affinity to when starting the component. The value should be a list of decimal values, including ranges. For example, `0-3,5,7-9` would set the affinity to CPUs 0,1,2,3,5,7,8 and 9. If empty then no CPU affinity is set and the component may be scheduled by the kernel onto any available CPU cores.

* `logfile` (Default: `gateway.log`)

  The file name of the component log file, relative to the `home` directory or an absolute path.

* `logdir` (Default: Unset)

  If set, it is used as the directory for the log file above. If not set (the default) then the `home` directory of the instance is used.

* `port` (Default: First available from `7038-7039,7100-`)

  The default port to listen on. The actual default is selected from the first available port in the range defined in `gateway::ports` in the program settings. If TLS is enabled, which is the default, then the base port is 7038 and 7039 is not selected. If TLS is not enabled then the base port is 7039. If you have multiple Gateways running on the same server then the `geneos add` and `geneos deploy` commands, amongst others, will automatically select the next available port in the range.

  The port range is defined in the top-level configuration as `gateway::ports` and defaults to `7038-7039,7100-`. You can change this using `geneos config set TYPE::ports="..."`. See the `geneos config` command for more details.

* `autostart` (Default: `true`)

  Gateway instances are set to be started with the default `geneos start` command. Setting `autostart` to false is different to using `geneos disable` to stop an instance from running. This can be used for instances that only need to be run occasionally or manually, for example a load monitoring Gateway instance. To start a Gateway that has `autostart` set to false you must give both the type and the name to the `geneos start` command, for example `geneos start gateway example2`.

* `protected` (Default: `false`)

  If `true` then the instance is protected from being changed or deleted by the `geneos start`, `geneos stop`, `geneos restart` or `geneos delete` and similar commands. This is useful for critical instances that should not be accidentally modified or removed. When an instance is protected, any attempt to change or delete it using the above commands will result in an error message unless the command is run with the `--force` option.

  This is different to using `geneos disable` to stop an instance from running. This can be used for instances that should not be changed or deleted, for example a production Gateway instance.

* `env` (Default: Empty)

  Environment variables set for the start-up of the Gateway are stored as an array of `NAME=VALUE` pairs. They should be set and unset using `geneos set -e` and `geneos unset -e` respectively to ensure consistency.

### TLS Parameters

* `tls::certificate` (Default: `${config:home}/gateway.pem`)
* `tls::privatekey` (Default: `${config:home}/gateway.key`)
* `tls::verify` (Default: `false`)
* `tls::ca-bundle` (Default: `${GENEOS_HOME}/tls/ca-bundle.pem`)
* `tls::minimumversion` (Default: `1.2`)

These parameters control TLS for the Gateway. TLS is enabled by default with the certificate and private key files in the instance home directory. If `tls::verify` is set to `true` then the Gateway will verify the remote endpoints it connects to, using the trusted roots in `tls::ca-bundle`. If `tls::verify` is set to `true` but the `tls::ca-bundle` file does not exist then the verification chain is set to an appropriate system default, which is seleected from a list of defaults for typical Linux systems.

If `tls::verify` is set to `true` but the `tls::ca-bundle` file does not exist then the verification chain is set to an appropriate system default, which is seleected from a list of defaults for typical Linux systems.

Deprecated parameters for TLS are also supported for backwards compatibility but should not be used in new configurations. If you are upgrading from an older version of `cordial` there is a `geneos tls migrate` command to help you. These deprecated parameters are (top-level pparameters, not under `tls`):

* `certificate`
* `privatekey`
* `certchain`
* `use-chain`

### Optional Parameters

* `options` (Default: Unset)

  A space separated set of additional options to append to the command line of the Gateway. For example, when you create a "demo" environment using `geneos init demo` the Gateway gets a `option` of `-demo`. The contents are split on space before being passed as individual arguments; this means that it is not possible to use arguments containing spaces, such as a file path.

  To pass extra parameters to the Gateway just once please see the `--extra`/`-x` option of the `geneos start`, `geneos restart` and `geneos deploy` commands.

## Usage

```text
geneos set [flags] [TYPE] [NAME...] [KEY=VALUE...]
```

### Options

```text
  -k, --keyfile KEYFILE               keyfile to use for encoding secrets
                                      default is instance configured keyfile,
                                      or user keyfile if not used by the instance type
  -s, --secure NAME[=VALUE]           encode a secret for NAME, prompt if VALUE not supplied, using a keyfile
  -e, --env NAME=VALUE                Environment variable for instance start-up
                                      (Repeat as required)
  -E, --secureenv NAME[=VALUE]        encode a secret for env var NAME, prompt if VALUE not supplied, using a keyfile
  -i, --include PRIORITY:[PATH|URL]   An include file in the format PRIORITY:[PATH|URL]
                                      (Repeat as required, gateway only)
  -g, --gateway HOSTNAME:PORT         A gateway connection in the format HOSTNAME:PORT
                                      (Repeat as required, san and floating only)
  -a, --attribute NAME=VALUE          Attribute in the format NAME=VALUE
                                      (Repeat as required, san only)
  -t, --type NAME                     A type NAME
                                      (Repeat as required, san only)
  -v, --variable [TYPE:]NAME=VALUE    A variable in the format [TYPE:]NAME=VALUE
                                      (Repeat as required, san only)
      --allow-root                    allow running as root (not recommended)
  -G, --config string                 config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME                 Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## Examples

```bash
geneos set gateway MyGateway licdsecure=false
geneos set infraprobe -e JAVA_HOME=/usr/lib/java8/jre -e TNS_ADMIN=/etc/ora/network/admin
geneos set -s secret netprobe local1
geneos set netprobe cloudapps1 -e SOME_CLIENT_ID=abcde -E SOME_CLIENT_SECRET

```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
