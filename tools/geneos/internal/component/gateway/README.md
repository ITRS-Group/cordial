# `geneos` Gateways

A `gateway` instance is a Geneos Gateway.

<https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_introduction.htm>

## Configuration

### Standard Parameters

Standard parameters always have values, using the defaults if not set in the configuration file.

#### `autostart`

Gateway instances are set to be started with the default `geneos start` command. Setting `autostart` to false is different to using `geneos disable` to stop an instance from running. This can be used for instances that only need to be run occasionally or manually, for example a load monitoring Gateway instance.

> Default: `true`

#### `binary`

The Gateway program filename. Should not be changed.

> Default: `gateway2.linux_64`

#### `gatewayname`

The Gateway's name can be different to the instance name. This is used in the default templates, under the Operating Environment created in `instance.setup.xml`

> Default: Instance Name

#### `home`

This parameter is special in that even though it can be changed it is re-evaluated based on the instance's directory

> Default: `${GENEOS_HOME}/gateway/gateways/NAME`  

#### `install`

The installation directory for Gateway releases

> Default: `${GENEOS_HOME}/packages/gateway`
    
#### `libpaths`

This parameter is combined with any `LD_LIBRARY_PATH` environment variable.

> Default: `${config:install}/${config:version}/lib64:/usr/lib64`

#### `licdhost`

> Default: `localhost`

#### `licdport`

> Default: `7041`

#### `logfile`

> Default: `gateway.log`

The file name of the Gateway log file.

#### `name`

> Default: Instance Name

#### `port`

The default port to listen on. The actual default is selected from the first available port in the range defined in `GatewayPortRange` in the program settings.

> Default: First available from `7038-7039,7100+`

#### `program`

The full path to the Gateway executable. The items in the default of the form `${config:NAME}` refer other configuration parameters above.

> Default: `${config:install}/${config:version}/${config:binary}`

#### `setup`

The Gateway setup file. This should normally not be changed.

> Default: `${config:home}/gateway.setup.xml`

#### `version`

The version of the Gateway in the the `install` directory above. This is normally the name of a symbolic version which is maintained as a link to a real installation version directory. You can create new symbolic version or tie an instance to an exact installed version.

> Default: `active_prod`

### Special Parameters

#### `config`

Parameters under the `config` section are related to the instance configuration handling and are not used for control of the Gateway environment.

##### `rebuild`

> Default: `initial`

The `rebuild` parameter controls how the instance responds to the `geneos rebuild` command. See below for more details.

##### `template`

> Default: `gateway.setup.xml.gotmpl`

The `template` parameter controls which template file is used to build the gateway setup file when `geneos rebuild` is run.`

#### `env`

Environment variables set for the start-up of the Gateway are stored as an array of `NAME=VALUE` pairs. They should be set and unset using `geneos set -e` and `geneos unset -e` respectively to ensure consistency.

#### `includes`

A list of include files to be used when building the Gateway setup file from templates. See below.

### Other Parameters

#### `protected`

> Default: `false`

#### `certificate` / `privatekey`

If defined these settings are filename or paths to TLS certificate and private key files in PEM format, respectively. When they are defined the Gateway is started with the appropriate secure options, and if the listening port is the default (7039) then it is updated to 7038 if and when the setup files are rebuilt.
    
With TLS initialised all new Gateway instances are created with certificates and private keys automatically.

> Defaults, if created: `gateway.pem` / `key.pem`

#### `logdir`

If set, it is used as the directory for the log file below. If not set (the default) then the `home` directory of the instance is used.

> Default: `${config:home}`

#### `keyfile`

#### `prevkeyfile`

#### `usekeyfile`

#### `user`

#### `options`

Additional options to pass on the command line to the Gateway. For example, when you create a "demo" environment using `geneos init demo` the Gateway gets a `option` of `-demo`.

#### `licdsecure`

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
