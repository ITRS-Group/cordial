# Notes

This information needs to be somewhere. Saved in this file for now.

## Directory Layout

The `geneos` configuration setting or the environment variable
`ITRS_HOME` points to the base directory for all subsequent operations.
The layout follows that of the original `gatewayctl` etc.

Directory structure / hierarchy / layout is as follows:

```text
/opt/itrs
├── fa2
│   └── fa2s
├── fileagent
│   └── fileagents
├── gateway
│   ├── gateway_config
│   ├── gateways
│   │   └── [gateway instance name]
│   ├── gateway_shared
│   └── templates
│       ├── gateway-instance.setup.xml.gotmpl
│       └── gateway.setup.xml.gotmpl
├── hosts
├── licd
│   └── licds
│       └── [licd instance name]
├── netprobe
│   └── netprobes
│       └── [netprobe instance name]
├── packages
│   ├── downloads
│   │   ├── geneos-gateway-6.0.0-linux-x64.tar.gz
│   │   ├── geneos-licd-6.0.0-linux-x64.tar.gz
│   │   ├── geneos-netprobe-6.0.2-linux-x64.tar.gz
│   │   └── geneos-web-server-6.0.0-linux-x64.tar.gz
│   ├── fa2
│   ├── fileagent
│   ├── gateway
│   │   ├── 6.0.0
│   │   └── active_prod -> 6.0.0
│   ├── licd
│   │   ├── 6.0.0
│   │   └── active_prod -> 6.0.0
│   ├── netprobe
│   │   ├── 6.0.2
│   │   └── active_prod -> 6.0.2
│   └── webserver
│       ├── 6.0.0
│       └── active_prod -> 6.0.0
├── san
│   ├── sans
│   └── templates
│       └── netprobe.setup.xml.gotmpl
└── webserver
    └── webservers
        └── [webserver instance name]
```

where:

* `fa2/` (Fix Analyser) contains settings & instance data related to the
  `fa2` component type.

  * `fa2/fa2s/` contains one sub-directory for each Fix Analyser
    instance named after the fa2 instance. These sub-directory will be
    used as working directories for the corresponding instances.

* `fileagent/` (File Agent for Fix Analyser) contains settings &
  instance data related to the `fileagent` component type.

  * `fileagent/fileagents/` contains one sub-directory for each File
    Agent instance named after the file agent instance. These
    sub-directory will be used as working directories for the
    corresponding instances.

* `gateway/` contains settings & instance data related to the `gateway`
  component type.

  * `gateway/gateway_config/` contains common Gateway configuration as
    include `XML` files.
  * `gateway/gateways/` contains one sub-directory for each Gateway
    instance named after the gateway instance. These sub-directories
    will be used as working directories for the corresponding gateway
    instances.
  * `gateway/gateway_shared/` contains shared Gateway data such as
    include `XML` files or scritped tools.
  * `gateway/templates/` contains Gateway configuration templates in the
    form of Golang XML templates.

* `hosts/` contains configurations for supporting control of Geneos
  component instances running on remote hosts.
* `licd/` (License Daemon) contains settings & instance data related to
  the `licd` component type.
  * `licd/licds/` contains one sub-directory for each licd instance
    named after the licd instance. This sub-directories will be used as
    working directories for the corresponding License Daemon (licd)
    instance.

* `netprobe/` contains settings & instance data related to the
  `netprobe` component type.
  * `netprobe/netprobes/` contains one sub-directory for each Netprobe
    instance named after the netprobe instance. These sub-directories
    will be used as working directories for the corresponding netprobe
    instances.

* `packages/` contains the Geneos binaries / software packages
  installed.
  * `packages/downloads/` contains files downloaded from the ITRS
    download portal, or the file repository used.
  * `packages/fa2/` contains one sub-directory for each version of Fix
    Analyser installed, as well as symlinks (e.g. `active_prod`)
    pointing to the current default version. These sub-directory will
    contain the corresponding binaries.
  * `packages/fileagent/` contains one sub-directory for each version of
    File Agent installed, as well as symlinks (e.g. `active_prod`)
    pointing to the current default version. These sub-directory will
    contain the corresponding binaries.
  * `packages/gateway/` contains one sub-directory for each version of
    Gateway installed, as well as a symlinks (e.g. `active_prod`)
    pointing to the current default version.  These sub-directory will
    contain the corresponding binaries.
  * `packages/licd/` contains one sub-directory for each version of
    License Daemon (licd) installed, as well as a symlinks (e.g.
    `active_prod`) pointing to the current default version. These
    sub-directory will contain the corresponding binaries.
  * `packages/netprobe/` contains one sub-directory for each version of
    Netprobe installed, as well as a symlinks (e.g. `active_prod`)
    pointing to the current default version. These sub-directory will
    contain the corresponding binaries.
  * `packages/webserver/` contains one sub-directory for each version of
    Webserver (for web dashboards) installed, as well as a symlinks
    (e.g. `active_prod`) pointing to the current default version. These
    sub-directory will contain the corresponding binaries.

* `san/` (Self-Announcing Netprobe) contains settings & instance data
  related to the `san` component type.
  * `san/sans/` contains one sub-directory for each Self-Announcing
    Netprobe instance named after the san instance. These
    sub-directories will be used as working directories for the
    corresponding san instances.
  * `san/templates/` contains Self-Announcing Netprobe configuration
    templates in the form of Golang XML templates.

* `webserver/` (Webserver for web dashbaords) contains settings &
  instance data related to the `webserver` component type.
  * `webserver/webservers/` contains one sub-directory for each
    Webserver instance named after the webserver instance. These
    sub-directories will be used as working directories for the
    corresponding Webserver instances.

The `bin/` directory and the default `.rc` files are **ignored**.
Please be careful in case you have customised anything in `bin/`.

As a very quick recap, each component directory will have a subdirectory
with the plural of the name (e.g. `gateway/gateways`) which will contain
subdirectories, one per instance, and these act as the configuration and
working directories for the individual processes. Taking an example
gateway called `Gateway1` the path will be:
`${ITRS_HOME}/gateway/gateways/Gateway1`.

This directory will be the working directory of the process and also
contain an `.rc` configuration file - if using the legacy scripts (e.g.
`gatewayctl`) - or a `.json` configuration file - if using the `geneos`
utility - as well as a `.txt` file to capture the `STDOUT` and `STDERR`
of the process.

There will also be an XML setup file and so on.


## `geneos` Components

### Instance Properties

**Note**: This section is incomplete and remains as work-in-progress.

| Property      | Previous Name | `licd`             | `gateway`          | `netprobe`         | `san`              | `fa2`              | `fileagent`        | `webserver`        | Description |
| --------      | ------------- | ------             | ---------          | ----------         | -----              | -----              | -----------        | -----------        | ----------- |
| `binary`      | `BinSuffix`   | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | Name of the binary file used to run the instance of the componenent TYPE. |
| n/a           | `TYPERoot`    | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Root directory for the TYPE. Ignored. |
| n/a           | `TYPEMode`    | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Process execution mode - baskground or foregbround. Ignored. |
| `home`        | `TYPEHome`    | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | Path to the instance's home directory, from where the instance component TYPE is started. |
| `install`     | `TYPEBins`    | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | Path to the directory where the binaries of the component TYPE are installed. |
| `libpaths`    | `TYPELibs`    | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | Library path(s) (separated by `:`) used by the instance of the component TYPE. |
| `logdir`      | `TYPELogD`    | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | Path to the dorectory where logs are to be written for the instance of the component TYPE. |
| `logfile`     | `TYPELogF`    | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | Name of the primary log file to be generated for the instance. |
| `name`        | `TYPEName`    | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | Name of the instance. |
| `options`     | `TYPEOpts`    | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | Additional command-line options to be used as part of the command line to start the instance of the component TYPE. |
| `port`        | `TYPEport`    | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | Listening port used by the instance. |
| `program`     | `TYPEExec`    | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | Absolute path to the binary file used to run the instance of the component TYPE. |
| `user`        | `TYPEUser`    | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | User owning the instance. |
| `version`     | `TYPEBase`    | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ | Version as either the name of the directory holding the component TYPE's binaries or the name of the symlink pointing to that directory. |
| Gateway Specific: |
| `gatewayname` | n/a *         | ❌ | ✔ | ❌ | ❌ | ❌ | ❌ | ❌ | Name of the gateway instance. This can be different to the instance name. |
| `licdhost`    | `GateLicH`    | ❌ | ✔ | ❌ | ❌ | ❌ | ❌ | ❌ | Name of the host where the license daemon (licd) to be used by the gateway instance is hosted. |
| `licdport`    | `GateLicP`    | ❌ | ✔ | ❌ | ❌ | ❌ | ❌ | ❌ | Port number of the license daemon (licd) to be used by the gateway instance. |
| `licdsecure`  | `GateLicS` *  | ❌ | ✔ | ❌ | ❌ | ❌ | ❌ | ❌ | Flag indicating whether connection to licd is secured by TLS encryption. |
| `keyfile`     | n/a           | ❌ | ✔ | ❌ | ❌ | ❌ | ❌ | ❌ | External keyfile for AES 256 encoding. |
| `prevkeyfile` | n/a           | ❌ | ✔ | ❌ | ❌ | ❌ | ❌ | ❌ | External keyfile for AES 256 encoding. |
| Webserver Specific: |
| `maxmem`      | `WebsXmx`     | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✔ | Java value for maximum memory for the Web Server (`-Xmx`) |
| TLS Settings: |
| `certificate` | `TYPECert` *  | 🔘 | 🔘 | 🔘 | 🔘 | 🔘 | ❌ | 🔘 | File containing a TLS certificate used for Geneos internal secure comms (TLS-encrypted). |
| `privatekey`  | `TYPEKey` *   | 🔘 | 🔘 | 🔘 | 🔘 | 🔘 | ❌ | 🔘 | File containing the privatye key associated with the TLS certificate `certificate`, used for Geneos internal secure comms (TLS-encrypted). |

Note: Settings in the `Previous Name`column with an `*` indicate those that were interim values during the development of the program and did not exist in the original `binutils` implementation.

Key:

| Checkmarks | `TYPE` labels in Pervious Name Column |
| ------ | ------ |
| ✔ - Supported and **required** | `gate` - Gateways |
| :radio_button: - Supports and optional | `licd` - License Daemons |
| :x: - Not support (and ignored) | `netp` - Netprobes |
| | `webs` - Web servers |
| | `FAgent` - File Agent |

In addition to the above simple properties there are a number of
properties that are lists of values and these values must be specific
formats.

* `env`

