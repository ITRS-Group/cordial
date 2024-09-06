# Self-Announcing Netprobe Configuration Server

To enable the fair distribution of monitoring load over multiple Geneos Gateways there is a requirement for a standalone server process that will serve requests for Self-Announcing Netprobe (SAN) configuration files based on live data from an inventory, combined with details in a configuration file. The server will respond to requests based solely on target URL to provide configuration files. Each location a configuration server process is deployed in may have it's own inventory provider and local configuration. The server process must be able to cope with reasonable load - for example bulk post-maintenance restarts - but this would have to be in-step with external API endpoints and any latencies as inventory data will not be cached to avoid stale data.

## Getting Started

To start using the `san-config` you have to, for each site:

1. Create or update the configuration file `san-config.yaml` to match local requirements
2. Deploy one or more instances of the `san-config`
3. Run Self-Announcing Netprobes with a `-setup URL` command line option to point to the correct set-up URL

Details for each step are below.

## Operation

The `san-config` program has two modes of operation, which can be accessed through sub-commands. The principle one is `server` to launch and run a continuous server process. The other sub-command is `host` and is available to assist in diagnostics and the manual creation of Netprobe setup files.

### `san-config server`

The most common mode of operation will be as a continually running server process. In this mode the program loads the configuration file, acquires inventories, checks available Geneos Gateways and responds on the configured URL to Netprobe setup requests. The program will auto reload most changes to it's configuration file unless configured not to do so. Changes to the configuration, except for the `server` section - see below, will be reflected in the other functions on their next scheduled operation; i.e. existing waits and timeouts will continue until they finish.

The program can also, optionally and enabled by default, serve a live AC2 remote connection file based on available gateways, but at the moment the Active Console will only load a remote configuration file at start-up or settings save so this is not as useful as it may appear. This should be addressed in a future AC2 release.

Note that the server process is largely stateless and you can deploy as many copies as required as long as the configured UUID Namespace is the same and the Netprobe can reach one of them on a fixed URL; This can be through round-robin or multiple A-record DNS or behind a load balancer.

The command line options available to the `server` command are:

```text
      --config string    path to configuration file
  -D, --daemon           Run as a daemon
  -L, --logfile string   Override configured log file path
  -N, --nowatch          Do not watch configuration file for changes
```

### `san-config host`

The `host` command runs the same operations as the `server` command but in a single-shot mode and instead of answering a remote request it will output the configuration on standard output.

This command is useful to check that the configuration is valid and also to look-up specific hosts and component types.

The only command line option supported by the `host` command is `--config PATH` to override the path to the YAML configuration file.

### Inventories

The program currently supports one type of inventory, plain YAML "name: type" pairs. Multiple inventory sources can be loaded and merged using an index (or "side"). Inventories can be loaded from remote URLs or local files.

#### Inventory Sources

Inventories can be loaded from remote URLs (with limited authentication support) or from local files. For both types of source the path can include expandable values. These include all the name/value pairs under `inventory.mappings` and also any other supported expand options, including environment variables and encoded credentials. Additionally the special `${index}` expansion item is set to each value in the `inventory.indices` list and the inventory sourced from each subsequent location. See below for how these multiple inventories are handled for the supported types.

For remote sources there is support for two types of authentication; `header` and `basic`. The type `header` adds an arbitrary HTTP Header given the name and the value while the `basic` type encoded a username and password using the HTTP Basic Auth standards.

Examples:

```yaml
inventory:
  mappings:
    environment: PROD
    region: EMEA
  indices: [ a, b ]
  source: https://gitlab.com/api/v4/projects/123456/repository/files/inventories-${environment}%2F${environment}-${region}-${index}.json/raw?ref=main
  authentication:
    type: header
    header: PRIVATE-TOKEN
    value: ${enc:~/.config/geneos/keyfile.aes:+encs+8F8F1FCACB5EBED9FE99E76291F88F38349120EC94EA9AA8077F0D0D1B11791B}
```

In this example, with the `indices` set to "a" and "b", the two URLs accessed would be:

```text
https://gitlab.com/api/v4/projects/123456/repository/files/inventories-PROD%2F$PROD-EMEA-a.json/raw?ref=main
https://gitlab.com/api/v4/projects/123456/repository/files/inventories-PROD%2F$PROD-EMEA-b.json/raw?ref=main
```

(Note, in this example, the GitLab API requires the use of HTTP URL encoding of '/' path separators - `%2F` to access files in directories)

The authentication header `value` field has been encoded using `geneos aes password` to opaque the value, which can only be decoded if you have the AES key-file referred to.

#### YAML Inventory

The default inventory type of `yaml` supports a simple mapping of the hostname to component type, e.g.

```yaml
host1: app0
host2: app1
host3: app2
```

Each component type (the right hand side) supports value expansion and the typical use may be to incorporate environment variables in the for `${env:APPNAME}` or, because there are no lookup tables (see the Expand docs) the plain `${APPNAME}` has the same effect.

The overall inventory used is the result of merging all hostname values together and if a hostname appears in multiple inventories then the one loaded last will be the one used.

### Component Types

Each configuration request results in a `hosttype` value, including for unknown hosts. This is used to lookup the component type and build the final SAN configuration file, along with the Gateway selection process detailed further below.

The `hosttype` can be overridden in the request and this is most typically used to indicate the request is from an infrastructure probe, e.g. `hardware`.

Each configured component type can specify values for:

* `probe-name` (required)
* Optional defaults for `attributes`, `types` and `variables`
* Managed Entities, each with:
  * `name` (required)
  * `attributes` (optional)
  * `types` (optional)
  * `variables` (optional)

In addition you can also specify global defaults for `attributes`, `types` and `variables` that apply to all component types in the reserved `components.defaults` section.

There is a special `components.unknown` type that is used when a request does not match any known inventory entry. Combined with the `geneos.fallback-gateway` this allows for the collection of all unconfigured probes to be directed to a central location for attention.

### Gateway Selection

The configuration server will direct a SAN to the same gateway (or hot-standby pair of gateways) if the configuration and availability of Gateways has not changed, including across restarts of the program or the SAN processes.

This is done using SHA1 hashed UUIDs to build consistent values based on the Gateway and Probe names (the latter offering a grouping facility to direct related probes to the Gateway). The process is described below:

1. A fixed "namespace UUID" is taken from the configuration file. This value can be change at initial installation but should then be left unchanged for the life of the system. There is no security requirement to change this value at any time.
2. Each incoming request has the `geneos.sans.grouping` regular expression applied and the first capture group is used as the input to generate another UUID using the above namespace and the contents of the capture group. The example in configuration is `"^(.*?)[ab]?$"` which strips off any "a" or "b" suffix. If this grouping regular expression is not set then the whole hostname is used.
    * If the incoming hostname is not found in the inventory then the Gateway selection process is cut short and the SAN is directed to the Gateway set configured in `geneos.fallback-gateway` to allow for unknown SANs to be seen and dealt with. This inventory lookup is done using the full hostname and not the result of the grouping regular expression.
3. The UUID generate in step 2 is then used as another namespace UUID against each "live" Gateway name (or primary `host:port` is a name is not defined) to generate a list of UUIDs, one per Gateway set.
4. The resulting list is used to sort the Gateways based on the UUID value and the first Gateway set returned.

As the namespace UUID (the seed value, if it's easier to visualise) is both unique and consistent for each grouped hostname the resulting UUID is effectively a securely hashed value for each live Gateway set.

The list of Gateways to use is based on a "liveness" test to a known URL on each Gateway. Both primary and standby Gateways are checked and as long as either responds then that Gateway set is considered available. This liveness endpoint has been available in all Gateways since 5.10 and is always enabled. The frequency of these tests and the timeout period are configurable.

## Configuration Reference

A configuration file, plus built-in defaults, drives the operation of the program. If not given explicitly with the `--config` option, the program looks for a configuration file in the following directories - the first one found is used:

1. the current directory
2. `${HOME}/.config/geneos`
3. `/etc/geneos`

The configuration file, by default, is `san-config.yaml`. If the program is renamed, for example to `serve-san-configs.linux`, then the base name (the file name with any extension removed) of the program is used as the base name of the configuration file - which would be `serve-san-configs.yaml` from the example. Symbolic links to the program are followed to obtain the underlying name of the binary.

The program has embedded defaults for many parameters and the default configuration is shown at the end of this README.

### Configuration Sections

The configuration file is in sections:

* `server`

    The config server program's listener details. This section is **NOT** reloaded automatically if the configuration file is changed.

* `geneos`

    A list of Geneos gateways to distribute SANs over. All Gateways can come in pairs and are otherwise considered the same with no differentiation of scale or capabilities. There is also the option for a "fallback" gateway pair for SANs that do not match any loaded inventory.

* `inventory`

    A list of inventories to load along with authentication and other details

* `components`

    A list of components and their defaults for the SAN setup

### Configuration Details

⚠ Until this README is further expanded, please see the comments in the default configuration file below for more details.

### Configuration Defaults

The embedded defaults are:

```yaml
...
```


