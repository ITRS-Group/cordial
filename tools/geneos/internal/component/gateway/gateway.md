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

### Instance Parameters

For general instance parameters, applicable to all component types, please see the documentation for the `geneos set` command, i.e. `geneos help set`.

The parameters described below are specific to the Gateway component.

* `gatewayname` (Default: Instance Name)

  The Gateway's name can be different to the instance name. This is used in the default templates, under the Operating Environment created in `instance.setup.xml`

* `setup` (Default: `${config:home}/gateway.setup.xml`)

  The Gateway setup file. If this is set to `none` or an empty string then no `-setup` option is passed on the command line. This allows for Centralised Config with Gateway Hub or Obcerv.

* `licdhost` (Default: `localhost`)
* `licdport` (Default: `7041`)
* `licdsecure` (Default: `false`)

  These three parameters control the connection to the license daemon. If `licdsecure` is set to `true` then the Gateway uses TLS to connect to the license daemon.

  When a new Gateway instance is created, `licdsecure` is set to `true` if TLS is enabled for the Gateway. If this is not correct for your environment then you should change it, either on the command like used to create the instance, i.e. `geneos add ... licdsecure=false` or `geneos deploy ... licdsecure=false`, or using `geneos set` later on.

* `usekeyfile` (Default: Depends on the version of the Gateway)

  Whether to use the key file for authentication with the license daemon. If `true` then the `keyfile` parameter is used to specify the path to the key file. If `false` then the `keyfile` parameter is ignored and not passed to the Gateway. The default depends on the version of the Gateway. For versions prior to 5.10.0 the default is `false`. For versions 5.10.0 and later the default is `true` and a keyfile is automatically created for the instance if it does not already exist.

* `keyfile` (Default: `${config:home}/gateway.aes` depending on version)
* `prevkeyfile` (Default: Empty)

  The file paths for the key file and previous key file used for AES256 encryption of secrets in the Gateway configuration files. These are only used if `usekeyfile` is `true`.

  If shared include files contain AES256 encrypted secrets then the same key file should be used across all Gateway instances that use those include files. This is because the key file contains the encryption key and if different key files are used then the secrets cannot be decrypted.

   The `prevkeyfile` is used when rotating keys. When a new key file is generated the old key file should be moved to a safe location and its path set in `prevkeyfile`. This allows the Gateway to decrypt secrets encrypted with the old key file while it is being rotated.

### Optional Parameters

* `insecureport` (Default: Unset)

  An optional additional port to listen on for non-TLS connections. This allows the Gateway to support both TLS and non-TLS connections at the same time.

* `snapshot::username` (Default: Unset)
* `snapshot::password` (Default: Unset)

  Optional username and password for the `snapshot` command to use when connecting to the Gateway REST API. These are used as defaults for the `snapshot` command and can be overridden on the command line by the `--user`/`-u` option. If these parameters are not set and no credentials can be found in the credentials file then the user will be prompted for a username and password when running the `snapshot` command.

### Configuration Parameters for `geneos rebuild`

When creating a new Gateway instance two XML setup files are created.

An `instance.setup.xml` include file is created and contains settings synthesised from the instance configuration. It is always updated when using the `geneos rebuild` command. This file is rebuilt regardless of the `config::rebuild` parameter.

A default `gateway.setup.xml` file is also created from the template(s) installed in the `gateway/templates` directory. By default this file is only created once but can be re-created using the `rebuild` command with the `-F` option if required. In turn this can also be protected against by setting the Gateway configuration setting `config::rebuild` to `never`.

* `config::rebuild` (Default: `initial`)

  The `rebuild` parameter controls how the instance responds to the `geneos rebuild` command. See below for more details.

* `config::template` (Default: `gateway.setup.xml.gotmpl`)

  The `template` parameter controls which template file is used to build the gateway setup file when `geneos rebuild` is run.

* `includes` (Default: Empty)

  A list of include files to be used when building the Gateway setup file from templates.

  Gateways support the setting of Include files for use in templated configurations. These are set similarly to the `-e` parameters:

  ```text
  geneos set gateway example2 -i 100:/path/to/include
  ```

  The setting value is `priority:path` and path can be a relative or absolute path or a URL. In the case of a URL the source is NOT downloaded but instead the URL is written as-is in the template output.

### Centralised Configuration

To use Centralised Configuration with either Gateway Hub or Obcerv, you should set the following parameters appropriately. They do not have defaults.

Also, you should set `setup` to either an empty value or the literal `none` to trigger Centralised Configuration on the Gateway.

* `gateway-hub` or `obcerv`

  One of these two parameters should be set to the URL of the Centralised Configuration store. Setting both is not a valid configuration.

* `app-key`

  To authenticate with an application key, set this parameter to the file path. Note that the application key file should be generated with the AES keyfile used by the Gateway **and** updated if the key file is changed.

* `kerberos-principal` and `kerberos-keytab`

  To authenticate using Kerberos, set these parameters as documented in the Gateway Installation Guide.