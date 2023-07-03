# `geneos set`

Set Instance Parameters

```text
geneos set [flags] [TYPE] [NAME...] [KEY=VALUE...]
```

Set one or more configuration parameters for matching instances.

Set will also update existing parameters, including setting them to empty values. To remove a parameter use the `geneos unset` command instead.

The command supports simple parameters given as `KEY=VALUE` pairs on the command line as well as options for structured or repeatable keys. Each simple parameter uses a case-insensitive `KEY`, unlike the options below.

Parameters can be encoded so that secrets do not appear in plain text in configuration files. Use the `--secure`/`-s` option with a parameter name and optional plaintext value. If no value is given then you will be prompted to enter the secret.

Environment variables can be set using the `--env`/`-e` option, which can be repeated as required, and the argument to the option should be in the format NAME=VALUE. An environment variable NAME will be set or updated for all matching instances under the configuration key `env`. These environment variables are used to construct the start-up environment of the instance. Environments can be added to any component TYPE.

Environment variables can be encoded so that secrets do not appear in plain text in configuration files. Use the `--secureenv`/`-E` option with a variable name and optional plaintext value. If no value is given then you will be prompted to enter the secret.

Include files (only used for Gateway component TYPEs) can be set using the `--include`/`-i` option, which can be repeated. The value must me in the form `PRIORITY:PATH/URL` where priority is a number between 1 and 65534 and the PATH is either an absolute file path or relative to the working directory of the Gateway. Alternatively a URL can be used to refer to a read-only remote include file. As each include file must have a different priority in the Geneos Gateway configuration file, this is the value that should be used as the unique key for updating include files.

Include file parameters are passed to templates (see `geneos rebuild`) and the template may or may not add additional values to the include file section. Templates are fully configurable and may not use these values at all.

For SANs and Floating Netprobes you can add or update Gateway connection details with the `--gateway`/`-g` option. These are given in the form `HOSTNAME:PORT`. The `HOSTNAME` can also be an IP address and is not the same as the `geneos host` command labels for remote hosts being managed, but the actual network accessible hostname or IP that the Gateway is listening on. This option can also be repeated as necessary and is applied to the instance configuration through templates, see `geneos rebuild`.

Three more options exist for SANs to set Attributes, Types and Variables respectively. As above these options can be repeated and will update or replace existing parameters and to remove them you should use `geneos unset`. All of these parameters depend on SAN configurations being built using template files and do not have any effect on their own. See `geneos rebuild` for more information.

Attributes are set using `--attribute`/`-a` with a value in the form `NAME=VALUE`.

Types are set using `--type`/`-t` and are just the NAME of the type. To remove a type use `geneos unset`.

Variables are set using `--variable`/`-v` and have the format [TYPE]:NAME=VALUE, where TYPE in this case is the type of content the variable stores. The supported variable TYPEs are: (`string`, `integer`, `double`, `boolean`, `activeTime`, `externalConfigFile`). These TYPE names are case sensitive and so, for example, `String` is not a valid variable TYPE. Other TYPEs may be supported in the future. Variable NAMEs must be unique and setting a variable with the name of an existing one will overwrite not just the VALUE but also the TYPE.

Future releases may add other special options and also may offer a simpler way of configuring SANs and Floating Netprobes to connect to Gateway also managed by the same `geneos` program.

### Options

```text
  -k, --keyfile KEYFILE               keyfile to use for encoding secrets
                                      default is instance configured keyfile
  -s, --secure NAME[=VALUE]           encode a secret for NAME, prompt if VALUE not supplied, using a keyfile
  -e, --env NAME=VALUE                An environment variable for instance start-up
                                      (Repeat as required)
  -E, --secureenv NAME[=VALUE]        encode a secret for env var NAME, prompt if VALUE not supplied, using a keyfile
  -i, --include PRIORITY:[PATH|URL]   An include file in the format PRIORITY:[PATH|URL]
                                      (Repeat as required, gateway only)
  -g, --gateway HOSTNAME:PORT         A gateway connection in the format HOSTNAME:PORT
                                      (Repeat as required, san and floating only)
  -a, --attribute NAME=VALUE          An attribute in the format NAME=VALUE
                                      (Repeat as required, san only)
  -t, --type NAME                     A type NAME
                                      (Repeat as required, san only)
  -v, --variable [TYPE:]NAME=VALUE    A variable in the format [TYPE:]NAME=VALUE
                                      (Repeat as required, san only)
```

## Examples

```bash
geneos set gateway MyGateway licdsecure=false
geneos set infraprobe -e JAVA_HOME=/usr/lib/java8/jre -e TNS_ADMIN=/etc/ora/network/admin
geneos set -p secret netprobe local1
geneos set ...

```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
