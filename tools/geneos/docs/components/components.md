# `geneos` Components

## Component Types

The following component types (and their aliases) are supported:

* **`gateway`** - or `gateways`

  A Geneos Gateway

* **`netprobe`** - or `netprobes`, `probe` or `probes`

* **`san`** - or `sans`

* **`floating`** - or `float`

* **`ca3`** - `collection-agent`, `collector` or `ca3s`

* **`licd`** - or `licds`

* **`webserver`** - or `webservers`, `webdashboard`. `dashboards`

* **`fa2`** - or `fixanalyser`, `fix-analyser`

* **`fileagent`** - or `fileagents`

* `any` (which is the default)

The first name, in bold, is also the directory name used for each type.
These names are also reserved words and you cannot configure (or expect
to consistently manage) components with those names. This means that you
cannot have a gateway called `gateway` or a probe called `probe`. If you
do already have instances with these names then you will have to be
careful migrating. See more below.

Each component type is described below along with specific component options.

**Note** This section is not yet complete, apologies.

## Instance Properties

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

