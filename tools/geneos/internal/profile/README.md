# `geneos` Profile Configuration

Profiles are configured through YAML files. This guide describes the meaning of the various parameters.

## General Profile Parameters

The top-level key must be `profiles`. Under this key, each profile is defined by a unique name and contains that profiles configuration.

For example:

```yaml
profiles:
  all:
    _short: "All components profile"
    _description: |
      The `all` profile is a base profile that includes all components
      with default settings.
    gateways:
      - name: Gateway1
        params:
          - port=7039
```

Each profile should include the following common parameters:

- `_short`: A brief description of the profile.
- `_description`: A detailed description of the profile.

Next. any keys under the name of the profile, without a leading underscore as above, are considered to be component configurations, such as `gateway`, `licd`, `netprobe`, or `webserver`. Any alias is also accepted, so `gateways`, `probe` or `webservers` are all valid. Under each component you can define any number of instances each with its own configuration parameters.

For example, to define a profile with one gateway and one netprobe you can do this:

```yaml
profiles:
  myprofile:
    gateway:
      name: Gateway1
      params:
        - port=7039
    netprobe:
      name: Netprobe1
      params:
        - port=7036
```

If you need more than one of a component type then just make sure you use YAML list format (indented, with the first key preceeded by a dash `-`), for example:

```yaml
profiles:
  myprofile:
    gateway:
      - name: Gateway1
        params:
          - port=7039
      - name: Gateway2
        params:
          - port=7040
```

## Component Configuration

### Supported Expansion Macros

Most parameter values can make use of expansion macros, which allow dynamic substitution of values based on the runtime environment or other profile settings.

The following are supported:

- `${env:NAME}`
  Expands to the value of the environment variable NAME

- `${option:EXPR}`
  Expands to the value of the `goval` expression which can use all the command line flags as well as configuration items as variables. For example, to use the value given to the `--port` flag:

  ```yaml
  - port=${option:port}
  ```

  Or, for a more complex expression which checks the `--insecure` flag:

  ```yaml
  - port=${option:insecure?8080:8443}
  ```

  Note that only those command-line flags that have been defined and passed to the program will be available for use in the `flag` expansion macros. You cannot add arbitrary flags for use in the expression. This may change in a future release.

- `${config:parameter}`
  Expands to the value of the configuration parameter `parameter` defined for an instance being iterated over in a `for-each` expansion. See more below.

  For example, if you have a `for-each` expansion iterating over gateways and each gateway has a `gatewayname` parameter, you can reference it like this:

  ```yaml
  - GATEWAY=${config:gatewayname}
  ```

- `${expr:EXPRESSION}`
  Expands to the result of evaluating the `goval` expression `EXPRESSION`. All configuration items (as for `${config:...}`) and environment variables (as for `${env:...}`) are available as variables within the expression. See `${option:...}` for similar support for command line flags.

- `${replace:params:/PATTERN/TEXT/}`
  Performs a regular expression replacement on the first non-empty configuration parameter specified in `params`. The `PATTERN` is the regular expression to match, and `TEXT` is the replacement text. The separator `/` can be any character except `:`, but must be used consistently to separate the pattern and the replacement text. 

- `${select:params:DEFAULT}`
  Selects the first non-empty parameter specified in `params`. If none of the parameters are set or all are empty, it returns the `DEFAULT` value. The `params` can be a list of parameter names separated by colons.

The external lookup macros are disabled. You cannot use `${file:...}` or similar external lookup macros.

### Common Parameters

These are common parameters that can be defined for any instance type within a profile.

- `name`
  The name of the instance. This is typically used to uniquely identify the instance within the profile.
  
- `env`
  Defined environment variables for the instance. This is a list of `NAME=VALUE` pairs, e.g

  ```yaml
  - env:
      - PATH=/usr/local/bin
      - HOME=/home/user
  ```

- `params`

  Defined parameters for the instance. This is a list of `KEY=VALUE` pairs, e.g

  ```yaml
  - params:
      - port=7039
      - host=localhost
  ```

### Gateway Specific Parameters

- `gateway-name`
  Set the Gateway's internal name. This can be different to the instance `name` defined above. These two settings are equivalent:

  ```yaml
  gateway-name: ${config:name}
  params:
    - gatewayname=${config:name}
  ```

- `licd-host`
  The hostname or IP address of the License Daemon (licd) that the Gateway should connect to. These two settings are equivalent:

  ```yaml
  licd-host: ${config:licdhost}
  params:
    - licdhost=${config:licdhost}
  ```
  
- `licd-port`
  The port number of the License Daemon (licd) that the Gateway should connect to. These two settings are equivalent:

  ```yaml
  licd-port: ${config:licdport}
  params:
    - licdport=${config:licdport}
  ```

- `licd-secure`
  Indicates whether the connection to the License Daemon (licd) should be secure (e.g., using TLS). These two settings are equivalent:

  ```yaml
  licd-secure: ${config:licdsecure}
  params:
    - licdsecure=${config:licdsecure}
  ```

  Note that if you do not set this value then the connection type will be decided by heuristics applied by `geneos` at runtime.

- `includes`
  Specifies additional configuration files to include in the generated Gateway setup file. This allows you to modularize your configuration by splitting it across multiple files.

  The format for this is:

  ```yaml
  includes:
    100: /path/to/config/file1.xml
    200: Include Group 1 | Include Group 2 | /path/to/config/file2.xml
  ```

  The numeric key is the priority assigned to the include file and this must be unique. Lower numbers indicate higher priority and the default priority of the main setup file is `1`.

  The value associated with each numeric key can either be a path to a configuration file or a combination of groups and a path, separated by the `|` character. Spaces around the `|` separator are removed when processing the groups and the path. When multiple includes refer to the same groups they are sorted into those groups at configuration time using the default `geneos` gateway template.

- `variables`

  Defines variables that will be included in the `Operating Environment` section of the `instance.setup.xml` runtime file that is generated for the Gateway. These variables can be used to customize the runtime environment for the Gateway instance.

  The format for defining variables is:

  ```yaml
  variables:
    - name: variable1
      type: string
      value: value
    - name: variable2
      type: integer
      value: 1
  ```

  The support for variable types includes `string`, `integer`, `double`, `boolean`, `activeTime`, `externalConfigFile`, and `secret`. If the type is not specified, it defaults to `string`.

### Self-Announcing Netrobe Specific Parameters

A Self-Announcing Netrobe (SAN) component can be configured with one or more managed entities, each representing an instance of the component within the monitoring environment. There is a "primary" managed entity, defined using the top-level `types`, `attributes` and `variables` settings and then further managed entities can be defined under the `managed-entities` section, which supports iterating over a list of instances matching the specified criteria. If no primary managed entity is defined then only those from the `managed-entities` section will be included. The choice of including a managed entity is based on the `types` settings, as without Types an entity defines no monitoring.

- `gateways`
  A list of gateways that the SAN should connect to. Each gateway should be in the form `HOST:PORT`. This is typically specified as:

  ```yaml
  gateways:
    - ${flag:gateway}
  ```

  Here, the command line option `--gateway` is used to specify the gateway to connect to and can be repeated. The value is processed and if there are multiple values on the command line, they are all included in the list of gateways. Also, if no gateways are specified on the command line, the `geneos` command will use `localhost:7038` or `localhost:7039` as the default gateway, depending on TLS being enabled or not, respectively.

  The choice of using TLS (secure) connections or not is determined by the SAN template and the configuration of the gateways specified. The logic, at the time of writing, in Go template format is:

  ```gotmpl
  {{- $secure := or $data.certchain (and $data.tls (or $data.tls.secure (index $data.tls "ca-bundle") $data.tls.verify)) }}
  ```

  Note: it is not important to understand the above at this time, it is here for reference only.

- `types`

  A list of Geneos Gateway Types that the SAN should request configuration for from the Gateway. If there are no types speficied then the `attributes` and `variables` sections will be ignored as without types defined there is no monitoring configuration to apply them to.

  Types are defined as a list of strings, e.g.:

  ```yaml
  types:
    - Level 1 - Infrastructure
    - Level 2 - Basic Application
  ```

- `attributes`

  A list of attributes that should be applied to the primary Managed Entity. Attributes are defined as a list of `NAME=VALUE` pairs, e.g.:

  ```yaml
  attributes:
    - ${flag:attribute}
    - ENVIRONMENT=Production
    - APPLICATION=Geneos
  ```

  In the above example, the repeatable command line flag `--attribute` is used to specify attributes, and it can be repeated multiple times to include multiple attributes. The settings are applied "in order" so if, as above, the same attribute is specified multiple times, the last occurrence will take precedence including overriding a command line flag `--attribute APPLICATION=Example`.

- `variables`

- `managed-entities`

  - `for-each`

    Defines a `for-each` expansion for iterating over a list of instances matching the specified component and the `match` and `ignore` criteria below. This allows you to dynamically generate configuration for multiple managed entities based on a common template.

    ```yaml
    - for-each: gateway
      name: ${config:name} Gateway
      attributes:
        - ${flag:attribute}
      types:
        - Level 2 - Geneos Gateway
    ```

  - `match`

  - `ignore`

  - `types`, `attributes`, `variables`

    These are the same as above, but additionally support `${config:<name>}` placeholders, which are replaced with the corresponding configuration values from any instances iterated over by the `for-each` expansion, for example:

    ```yaml
    - for-each: gateway
      name: ${replace:gatewayname:name:/ /_/} Gateway
      attributes:
        - ${flag:attribute}
        - GATEWAY=${replace:gatewayname:name:/ /_/}
      variables:
        - type: string
          name: logfile
          value: ${config:home}/${config:logfile}
    ```

### Floating Netprobe Specific Parameters

- `gateways`
  A list of gateways that the SAN should connect to. Each gateway should be in the form `HOST:PORT`. This is typically specified as:

  ```yaml
  gateways:
    - ${flag:gateway}
  ```

  Here, the command line option `--gateway` is used to specify the gateway to connect to and can be repeated. The value is processed and if there are multiple values on the command line, they are all included in the list of gateways. Also, if no gateways are specified on the command line, the `geneos` command will use `localhost:7038` or `localhost:7039` as the default gateway, depending on TLS being enabled or not, respectively.

  The choice of using TLS (secure) connections or not is determined by the SAN template and the configuration of the gateways specified. The logic, at the time of writing, in Go template format is:
