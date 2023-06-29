# `geneos gateway`

Gateways

```text
geneos gateway
```

# `geneos` Gateways

A `gateway` instance is a Geneos Gateway.

<https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_introduction.htm>

## Configuration

* `binary`

    > Default: `gateway2.linux_64`

    The Gateway program filename. Should not be changed.

* `home`  

    > Default: `${GENEOS_HOME}/gateway/gateways/NAME`  

    This parameter is special in that even though it can be changed it is re-evaluated based on the instance's directory

* `install`

    > Default: `${GENEOS_HOME}/packages/gateway`
    
    The installation directory for Gateway releases

* `version`

    > Default: `active_prod`

    The version of the Gateway in the the `install` directory above. 

* `program`

    > Default: `${config:install}/${config:version}/${config:binary}`

    The full path to the Gateway executable. The items in the default of the form `${config:NAME}` refer other configuration parameters above.

* `logdir`

    > Default: none

    If set, it is used as the directory for the log file below. If not set (the default) then the `home` directory of the instance is used.

* `logfile`

    > Default: `gateway.log`

    The file name of the Gateway log file.

* `port`

    > Default: First available from `7038-7039,7100+`

    The default port to listen on. The actual default is selected from the first available port in the range defined in `GatewayPortRange` in the program settings.

* `libpaths`

    > Default: `${config:install}/${config:version}/lib64:/usr/lib64`

    This parameter is combined with any `LD_LIBRARY_PATH` environment variable.

* `gatewayname`

    > Default: Instance Name

* `setup`

    > Default: `${config:home}/gateway.setup.xml`

    The Gateway setup file.

* `autostart`

    > Default: `true`

* `protected`

    > Default: `false`


## Gateway templates

When creating a new Gateway instance a default `gateway.setup.xml` file is created from the template(s) installed in the `gateway/templates` directory. By default this file is only created once but can be re-created using the `rebuild` command with the `-F` option if required. In turn this can also be protected against by setting the Gateway configuration setting `config::rebuild` to `never`.

### Gateway variables for templates

Gateways support the setting of Include files for use in templated configurations. These are set similarly to the `-e` parameters:

```text
geneos gateway set example2 -i 100:/path/to/include
```

The setting value is `priority:path` and path can be a relative or absolute path or a URL. In the case of a URL the source is NOT downloaded but instead the URL is written as-is in the template output.

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
