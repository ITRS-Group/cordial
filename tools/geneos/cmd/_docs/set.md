Set one or more configuration parameters for matching instances.

Set will also update existing parameters, including setting them to empty values. To remove a parameter use the `geneos unset` command instead.

The command supports simple parameters given as `KEY=VALUE` pairs on the command line as well as options for structured or repeatable keys. Each simple parameter uses a case-insensitive `KEY`, unlike the options below.

You can also use `+=` or `+` to append values to an exist parameter (or create a parameter if it does not exist), e.g. `options+="-extra option"`. If the value starts with a dash it is assumed to be a new command line parameter and is appended with a space, otherwise it is appended as-is and you are responsible for ensuring the resulting parameter is correctly formatted, e.g. paths having ":" separators.

Parameters can be encoded so that secrets do not appear in plain text in configuration files. Use the `--secure`/`-s` option with a parameter name and optional plaintext value. If no value is given then you will be prompted to enter the secret.

Environment variables can be set using the `--env`/`-e` option, which can be repeated as required, and the argument to the option should be in the format NAME=VALUE. An environment variable NAME will be set or updated for all matching instances under the configuration key `env`. These environment variables are used to construct the start-up environment of the instance. Environments can be added to any component TYPE.

Environment variables can be encoded so that secrets do not appear in plain text in configuration files. Use the `--secureenv`/`-E` option with a variable name and optional plaintext value. If no value is given then you will be prompted to enter the secret.

Include files (only used for Gateway component TYPEs) can be set using the `--include`/`-i` option, which can be repeated. The value must me in the form `PRIORITY:PATH/URL` where priority is a number between 1 and 65534 and the PATH is either an absolute file path or relative to the working directory of the Gateway. Alternatively a URL can be used to refer to a read-only remote include file. As each include file must have a different priority in the Geneos Gateway configuration file, this is the value that should be used as the unique key for updating include files.

Include file parameters are passed to templates (see `geneos rebuild`) and the template may or may not add additional values to the include file section. Templates are fully configurable and may not use these values at all.

For SANs and Floating Netprobes you can add or update Gateway connection details with the `--gateway`/`-g` option. These are given in the form `HOSTNAME:PORT`. The `HOSTNAME` can also be an IP address and is not the same as the `geneos host` command labels for remote hosts being managed, but the actual network accessible hostname or IP that the Gateway is listening on. This option can also be repeated as necessary and is applied to the instance configuration through templates, see `geneos rebuild`.

Three more options exist for SANs to set Attributes, Types and Variables respectively. As above these options can be repeated and will update or replace existing parameters and to remove them you should use `geneos unset`. All of these parameters depend on SAN configurations being built using template files and do not have any effect on their own. See `geneos rebuild` for more information.

Attributes are set using `--attribute`/`-a` with a value in the form `NAME=VALUE`.

Types are set using `--type`/`-t` and are just the NAME of the type. To remove a type use `geneos unset`.

Geneos User Variables are set using `--variable`/`-v` and have the format `[TYPE]:NAME=VALUE`, where `TYPE` in this case is the type of content the variable stores. The supported variable `TYPEs` are: (`string`, `integer`, `double`, `boolean`, `activeTime`, `externalConfigFile`). These `TYPE` names are case sensitive and so, for example, `String` is not a valid variable `TYPE`. Other TYPEs may be supported in the future. Variable `NAMEs` must be unique and setting a variable with the name of an existing one will overwrite not just the VALUE but also the `TYPE`.

Future releases may add other special options and also may offer a simpler way of configuring SANs and Floating Netprobes to connect to Gateway also managed by the same `geneos` program.
