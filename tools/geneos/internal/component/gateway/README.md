A `gateway` component is an instance of a [Geneos Gateway](https://docs.itrsgroup.com/docs/geneos/current/processing/gateway_introduction/index.html).

## Configuration

The Gateway instance configuration, as opposed to the runtime configuration of the Gateway itself, is stored in the instance configuration file. This is a JSON file which is created with the instance and updated when the `geneos set` and `geneos unset` commands are used to change parameters. The configuration file is stored in the instance directory as `gateway.json`. This file should not be edited directly but instead the `geneos set` and `geneos unset` commands should be used to change the configuration parameters.

In a future `cordial` release the configuration file may move to a YAML format for better readability but the JSON format would continue to be supported for backwards compatibility.

### Gateway Configuration Files

#### `gateway.setup.xml`

(Default: Created from template)

When a Gateway instance is created, a simple `gateway.setup.xml` file is created using the template file `${GENEOS_HOME}/gateway/templates/gateway.setup.xml.gotmpl`. This file is used as the setup file for the Gateway and is passed to the Gateway using the `-setup` option. The default template can be changed, per instance, by setting the `config::template` parameter. See the `config::rebuild` and `config::template` parameters below for more details.

#### `instance.setup.xml`

(Default: Created from template)

An `instance.setup.xml` file is created in the Gateway instance directory. This file is built using the template file `${GENEOS_HOME}/gateway/templates/instance.setup.xml.gotmpl` and is "included" by the `gateway.setup.xml` file and contains settings synthesised from the instance configuration. This file is always updated when using the `geneos set`, `geneos unset` and `geneos rebuild` commands, regardless of the `config::rebuild` parameter. If you add environment variables to the instance configuration then they are added to the `instance.setup.xml` file as variables with names of the form `_ENV_VAR_NAME` and string values. For example, if you set an environment variable of `FOO=BAR` then a variable named `_ENV_FOO` with value `BAR` is created in the `instance.setup.xml` file. Any variables you add to the instance configuration using `geneos set gateway GATEWAY -v type:name=value` are also added to the `instance.setup.xml` file as variables with their specified names, types and values. For example, if you set a variable of `NAME=VALUE` with type `string` then a variable named `NAME` with value `VALUE` and type `string` is created in the `instance.setup.xml` file. This behaviour can be customised by setting the `config::template` parameter to use a different template that does not include these variables or includes them in a different way.

### Standard Parameters

Standard parameters always have values, using a default if not changed in the configuration file.

In the examples below, `${GENEOS_HOME}` is the directory of the Geneos installation, which is normally `/opt/itrs/geneos` but can be set during initialisation, and values like `${config:PARAMETER}` are references to another configuration parameters, which are evaluated and replaced in the resulting value.

* `name` (Default: Instance Name)

  The name of the Gateway. This is used in the default templates, under the Operating Environment created in `instance.setup.xml`. It should not be changed. See also `gatewayname` below.

* `home` (Read Only: `${GENEOS_HOME}/gateway/gateways/${config:name}`)

  This parameter is read-only and is set based on the instance's directory. `${config:name}` is the instance name, not the Gateway name. This allows you to move the instance directory and have the `home` parameter update accordingly. It is used as the working directory for the Gateway process.

* `gatewayname` (Default: Instance Name)

  The Gateway's name can be different to the instance name. This is used in the default templates, under the Operating Environment created in `instance.setup.xml`

* `install` (Default: `${GENEOS_HOME}/packages/gateway`)

  The installation directory for Gateway releases

* `version` (Default: `active_prod`)

  The version of the Gateway in the the `install` directory above. This is normally the name of a symbolic version (the "basename") which is maintained as a link to a real installation version directory. You can create new symbolic version or tie an instance to an exact installed version. See the `geneos package install` and `geneos package update` commands for more details.

* `binary` (Default: `gateway2.linux_64`)

  The Gateway program filename. Should not be changed.

* `program` (Default: `${config:install}/${config:version}/${config:binary}`)

  The full path to the Gateway executable. The items in the default of the form `${config:NAME}` refer other configuration parameters above.

* `setup` (Default: `${config:home}/gateway.setup.xml`)

  The Gateway setup file. If this is set to `none` or an empty string then no `-setup` option is passed on the command line. This allows for Centralised Config with Gateway Hub or Obcerv.

* `libpaths` (Default: `${config:install}/${config:version}/lib64:/usr/lib64`)

  This parameter is combined with any `LD_LIBRARY_PATH` environment variable to create the `LD_LIBRARY_PATH` used when starting the Gateway. The default is the `lib64` directory of the Gateway installation version and the standard system library directory.

* `options` (Default: Empty)

  A space separated set of additional options to append to the command line of the Gateway. For example, when you create a "demo" environment using `geneos init demo` the Gateway gets a `option` of `-demo`. The contents are split on space before being passed as individual arguments; this means that it is not possible to use arguments containing spaces, such as a file path.

  To pass extra parameters to the Gateway just once please see the `--extra`/`-x` option of the `geneos start`, `geneos restart` and `geneos deploy` commands.

* `licdhost` (Default: `localhost`)
* `licdport` (Default: `7041`)
* `licdsecure` (Default: `false`)

  These three parameters control the connection to the license daemon. If `licdsecure` is set to `true` then the Gateway uses TLS to connect to the license daemon.

  When a new Gateway instance is created, `licdsecure` is set to `true` if TLS is enabled for the Gateway. If this is not correct for your environment then you should change it, either on the command like used to create the instance, i.e. `geneos add ... licdsecure=false` or `geneos deploy ... licdsecure=false`, or using `geneos set` later on.

* `logfile` (Default: `gateway.log`)

  The file name of the Gateway log file, relative to the `home` directory or an absolute path.

* `logdir` (Default: Unset)

  If set, it is used as the directory for the log file above. If not set (the default) then the `home` directory of the instance is used.

* `usekeyfile` (Default: Depends on the version of the Gateway)

  Whether to use the key file for authentication with the license daemon. If `true` then the `keyfile` parameter is used to specify the path to the key file. If `false` then the `keyfile` parameter is ignored and not passed to the Gateway. The default depends on the version of the Gateway. For versions prior to 5.10.0 the default is `false`. For versions 5.10.0 and later the default is `true` and a keyfile is automatically created for the instance if it does not already exist.

* `keyfile` (Default: `${config:home}/gateway.aes` depending on version)
* `prevkeyfile` (Default: Empty)

  The file paths for the key file and previous key file used for AES256 encryption of secrets in the Gateway configuration files. These are only used if `usekeyfile` is `true`.

  If shared include files contain AES256 encrypted secrets then the same key file should be used across all Gateway instances that use those include files. This is because the key file contains the encryption key and if different key files are used then the secrets cannot be decrypted.

   The `prevkeyfile` is used when rotating keys. When a new key file is generated the old key file should be moved to a safe location and its path set in `prevkeyfile`. This allows the Gateway to decrypt secrets encrypted with the old key file while it is being rotated.

* `port` (Default: First available from `7038-7039,7100-`)

  The default port to listen on. The actual default is selected from the first available port in the range defined in `gateway::ports` in the program settings. If TLS is enabled, which is the default, then the base port is 7038 and 7039 is not selected. If TLS is not enabled then the base port is 7039. If you have multiple Gateways running on the same server then the `geneos add` and `geneos deploy` commands, amongst others, will automatically select the next available port in the range.

  The port range is defined in the top-level configuration as `gateway::ports` and defaults to `7038-7039,7100-`. You can change this using `geneos config set gateway::ports="..."`. See the `geneos config` command for more details.

* `insecureport` (Default: Unset)

  An optional additional port to listen on for non-TLS connections. This allows the Gateway to support both TLS and non-TLS connections at the same time.

* `tls`

  * `tls::certificate` (Default: `${config:home}/gateway.pem`)
  * `tls::privatekey` (Default: `${config:home}/gateway.key`)
  * `tls::verify` (Default: `false`)
  * `tls::ca-bundle` (Default: `${GENEOS_HOME}/tls/ca-bundle.pem`)
  * `tls::minimumversion` (Default: `1.2`)

  These parameters control TLS for Gateway connections. If `tls::certificate` and `tls::privatekey` are set then TLS is enabled and the Gateway is started with the appropriate options. The default is to have TLS enabled with the certificate and private key files in the instance home directory. If `tls::verify` is set to `true` then the Gateway will verify the remote endpoints it connects to, using the trusted roots in `tls::ca-bundle`.

  If `tls::verify` is set to `true` but the `tls::ca-bundle` file does not exist then the verification chain is set to an appropriate system default, which is seleected from a list of defaults for typical Linux systems.

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

* `config::rebuild` (Default: `initial`)

  The `rebuild` parameter controls how the instance responds to the `geneos rebuild` command. See below for more details.

* `config::template` (Default: `gateway.setup.xml.gotmpl`)

  The `template` parameter controls which template file is used to build the gateway setup file when `geneos rebuild` is run.

* `env` (Default: Empty)

  Environment variables set for the start-up of the Gateway are stored as an array of `NAME=VALUE` pairs. They should be set and unset using `geneos set -e` and `geneos unset -e` respectively to ensure consistency.

* `includes` (Default: Empty)

  A list of include files to be used when building the Gateway setup file from templates.

### Centralised Configuration

To use Centralised Configuration with either Gateway Hub or Obcerv, you should set the following parameters appropriately. They do not have defaults.

Also, you should set `setup` to either an empty value or the literal `none` to trigger Centralised Configuration on the Gateway.

* `gateway-hub` or `obcerv`

  One of these two parameters should be set to the URL of the Centralised Configuration store. Setting both is not a valid configuration.

* `app-key`

  To authenticate with an application key, set this parameter to the file path. Note that the application key file should be generated with the AES keyfile used by the Gateway **and** updated if the key file is changed.

* `kerberos-principal` and `kerberos-keytab`

  To authenticate using Kerberos, set these parameters as documented in the Gateway Installation Guide.

## Gateway templates

When creating a new Gateway instance two setup files are created.

An `instance.setup.xml` include file is created and contains settings synthesised from the instance configuration. It is always updated when using the `geneos rebuild` command. This file is rebuilt regardless of the `config::rebuild` parameter.

A default `gateway.setup.xml` file is also created from the template(s) installed in the `gateway/templates` directory. By default this file is only created once but can be re-created using the `rebuild` command with the `-F` option if required. In turn this can also be protected against by setting the Gateway configuration setting `config::rebuild` to `never`.

### Gateway variables for templates

Gateways support the setting of Include files for use in templated configurations. These are set similarly to the `-e` parameters:

```text
geneos set gateway example2 -i 100:/path/to/include
```

The setting value is `priority:path` and path can be a relative or absolute path or a URL. In the case of a URL the source is NOT downloaded but instead the URL is written as-is in the template output.
