# `geneos` management tool

The `geneos` program will help you manage your Geneos environment on
Linux.

The online documentation for all the commands and options is [here](docs/geneos.md)

## Aims

* Make your life easier; at least the part managing Geneos
* Keep it simple through the [Principle of least
  astonishment](https://en.wikipedia.org/wiki/Principle_of_least_astonishment)
* Help you use automation tools with Geneos

## Basic Features

* Initialise a new installation with one command
* Adopt an existing installation that uses older tools
* Manage a group of servers with a single command
* Create and manage Geneos compatible AES256 encoded passwords
* Manage certificates for TLS connectivity between Geneos components
* Configure instance settings without editing files
* Download and install Geneos software, update components
* Simple bootstrapping of Self-Announcing Netprobes

## Getting Started

First download the pre-built binary or build from source. See
[Installation](#installation) for details.



For typical commands the command line will be like this:

`geneos COMMAND [flags] [TYPE] [NAMES...]`

* `COMMAND` is the one or two word name of the command
* `flags` are dash or double-dash prefixed options. Many of these
  options will require arguments and the details for each command will
  be in the help text.
* `TYPE` is the optional component type
* `NAMES` is an optional list of instance names. They can have an
  optional `@HOST` suffix which restricts the match to only the
  configured HOST.

The overall goal when choosing command names and their options has been
to make things more intuitive rather than less. This is not always
possible and so some options will be different between commands while
performing the same task; One examples is the `-f` (follow) flags for
`logs` which has similar functionality to the `-l` (logs) flag for
`init` commands.

### Legacy Command Emulation

If you run the program with a name ending in `ctl`, either through a
symbolic link or by copying the binary, then the legacy command syntax
is emulated in a simplistic way. This will allow for users or automation
scripts to continue working in their environment and get used to the
`geneos` command syntax gradually. The first half of the executable name
is mapped to the component type, so for example:

```bash
ln -s geneos gatewayctl
# this then runs ./geneos start gateway GW1
./gatewayctl GW1 start

ln -s geneos netprobectl
# this then runs ./geneos list netprobe
./netprobectl list
```

In general `TYPEctl NAME COMMAND` becomes `geneos COMMAND TYPE NAME`

The word `all` instead of a specific instance name is supported as
expected.

### Available Commands

Commands have their own automatically generated documents and are listed
here: [`geneos.md`](docs/geneos.md)

## Concepts & Terminology

This documentation and the program itself assumes familiarity with the
Geneos suite of products. Many of the key terms have been inherited from
earlier systems.

The specific types supported by this program are details in
[Component Types](#geneos-components).

### Geneos

[Geneos](https://www.itrsgroup.com/products/geneos) is a suite of
software products from [ITRS](https://www.itrsgroup.com/) that provide
real-time visibility of I.T. infrastructure and trading environments. It
uses a three-tier architecture to collect, process and present enriched
data to administrators.

### Components

A *component* is a type of software package and associated data. Each
component will typically be a software package from one of the
three-tiers mentioned above but can also be a derivative, e.g. a
Self-Announcing Netprobe is a component type that abstracts the special
configuration of either a vanilla Netprobe or, for example, the Fix
Analyser Netprobe.

The supported component types are listed in
[Component Types](docs/geneos.md).

### Instances

An *instance* is an independent copy of a component with a working
directory (`<top-level directory>/<component>/<component>s/<instance
name>`, e.g. `/opt/itrs/netprobe/netprobes/myNetprobe`), configuration
and other persistent files. Instances share read-only package
directories for the binaries and other files from the distribution for
the specific version being used.

### Hosts

*Hosts* are the locations that components are installed and
instantiated. There is always a *localhost*.

## Instance Protection

Individual instances can be protected again being stopped or deleted by
setting it to be protected.

```bash
geneos protect gateway IMPORTANT_GW
```

This also applies to almost any command that stops an instance, such as
the more obvious ones like `restart` but also `disable` and others. The
`copy` command, because it must be given the name for a source, does not
check this setting. For most commands that do check the protection
setting before running there is a `--force` flag to override the
protection. The `delete` command already requires that an instance be
disabled or called with the `--force` flag.

If you run `geneos delete host HOSTNAME` with the `--stop` flag to stop
instance on the remote host first, then the `protected` settings is also
checked and the command will terminate on the first error. This does
however mean that unprotected instances on that host may have been
stopped in the meantime.

The `update` command will not run if any protected instance is using the
base symlink about to be updated.

## Environment Settings

The `geneos` program uses the packages [Cobra](https://cobra.dev) and
[Viper](https://github.com/spf13/viper) (the latter via a wrapper
package) to provide the command syntax and configuration management.
There is full support for Viper's layered configuration for non-instance
settings, which means you can override global and user settings with
environment variables prefixed `ITRS_`, e.g. `ITRS_DOWNLOAD_USERNAME`
overrides `download.username`

## Instance Settings

Each instance has a configuration file. This is the most basic
expression of an instance. New instances that you create will have a
configuration file named after the component type plus the extension
`.json`. Older instances which you have adopted from older control
scripts will have a configuration file with the extension `.rc`

### Legacy Configuration Files

Historical (aka. legacy) `.rc` files have a simple format of the form

```bash
GatePort=1234
GateUser=geneos
```

Where the prefix (`Gate`) also encodes the component type and the suffix
(e.g. `Port`) is the setting. Any lines that do not contain the prefix
are treated as environment variables and are evaluated and passed to the
program on start. Lines that contain environment variables like
`${HOME}` will be expanded at run time. If the configuration is
migrated, either through an explicit `geneos migrate` command or if a
setting is changes through `geneos set` or similar then the value of the
environment variable will be carried over and continue to be expanded at
run-time. The `geneos show` command can be passed a `--raw` flag to show
the unexpanded values, if any.

While the `geneos` program can parse and understand the legacy `.rc`
files above it will never update them, instead migrating them to their
`.json` equivalents either when required or when explicitly told to
using the `migrate` command.

### JSON Configuration Files

The `.json` configuration files share common parameters as well as
component type specific settings. For brevity some of these parameters
are overloaded and have different meanings depending on the component
type they apply to.

While editing the configuration files directly is possible, it is best
to use the `set` and `unset` commands to ensure the syntax is correct.

### Special parameters

All instance types support custom environment variables being set or
unset. This is done through the `set` and `unset` commands below,
alongside the standard configuration parameters for each instance type.

Some component types, namely Gateways and SANs, support other special
parameters via other command line flags. See the help text or the full
documentation for the `set` and `unset` commands for more details.

To set an environment variable use this syntax:

```bash
geneos set netprobe example1 -e PATH_TO_SOMETHING=/file/path
```

If an entry already exists it is overwritten.

To remove an entry, use `unset`, like this

```bash
geneos unset netprobe example1 -e PATH_TO_SOMETHING
```

You can specify multiple entries by using the flag more than once:

```bash
geneos set netprobe example1 -e JAVA_HOME=/path -e ORACLE_HOME=/path2
```

Finally, if your environment variable value contains spaces then use
quotes as appropriate to your shell to prevent those spaces being
processed. In bash you can do any of these to achieve the same result:

```bash
geneos set netprobe example1 -e MYVAR="a string with spaces"
geneos set netprobe example1 -e "MYVAR=a string with spaces"
```

You can review the environment for any instance using the `show` command:

```bash
geneos show netprobe example1
```

Also. output is available from the `command` command to show what would
be run when calling the `start` command:

```bash
geneos command netprobe example1
```

#### General Command Flags & Arguments

```bash
geneos COMMAND [flags] [TYPE] [NAME...] [PARAM...]
```

Where:

* `COMMAND` - one of the configured commands
* `flags` - Both general and command specific flags
* `TYPE` - the component type
* `NAME` - one or more instance names, optionally including the remote server
* `PARAM` - anything that isn't one of the above

In general, with the exception of `COMMAND` and `TYPE`, all parameters
can be in any order as they are filtered into their types for most
commands. Some commands require arguments in an exact order.

As an example, these have the same meaning:

```bash
geneos ls -c gateway one two three
geneos ls gateway one -c two three
```

Reserved instance names are case-insensitive. So, for example,
"gateway", "Gateway" and "GATEWAY" are all reserved.

The `NAME` is of the format `INSTANCE@REMOTE` where either is optional.
In general commands will wildcard the part not provided. There are
special `REMOTE` names `@localhost` and `@all` - the former is, as the
name suggests, the local server and `@all` is the same as not providing
a remote name.

There is a special format for adding SANs in the form `TYPE:NAME@REMOTE`
where `TYPE` can be used to select the underlying Netprobe type. This
format is still accepted for all other commands but the `TYPE` is
silently ignored.

#### File and URLs

In general all source file references support URLs, e.g. importing
certificate and keys, license files, etc.

The primary exception is for Gateway include files used in templated
configurations. If these are given as URLs then they are used in the
configuration as URLs.

## Configuration Files

### General Configuration

* `/etc/geneos/geneos.json` - Global options
* `${HOME}/.config/geneos/geneos.json` - User options
* Environment variables ITRS_`option` - where `.` is replaced by `_`,
  e.g. `ITRS_DOWNLOAD_USERNAME`

General options are loaded from the global config file first, then the
user one and any environment variables override both files. The current
options are:

* `geneos`

The home directory for all other commands. See [Directory
Layout](#directory-layout) below. If set the environment variable
ITRS_HOME overrides any settings in the files. This is to maintain
backward compatibility with older tools. The default, if not set
anywhere else, is the home directory of the user running the command or,
if running as root, the home directory of the `geneos` or `itrs` users
(in that order). (To be fully implemented) This value is also set by the
environment variables `ITRS_HOME` or `ITRS_GENEOS`

* `download.url`

The base URL for downloads for automating installations. Not yet used.
If files are locally downloaded then this can either be a `file://`
style URL or a directory path.

* `download.username` `download.password`

  These specify the username and password to use when downloading
  packages. They can also be set as the environment variables, but the
  environment variables are not subject to expansion and so cannot
  contain Geneos encoded passwords (see below):

  * `ITRS_DOWNLOAD_USERNAME`
  * `ITRS_DOWNLOAD_PASSWORD`

* `snapshot.username` `snapshot.password`

  Similarly to the above, these specify the username and password to use
  when taking dataview snapshots. They can also be set as the
  environment variables, with the same restrictions as above:

  * `ITRS_SNAPSHOT_USERNAME`
  * `ITRS_SNAPSHOT_PASSWORD`

* `GatewayPortRange` & `NetprobePortRange` & `LicdPortRange`

...

### Component Configuration

For compatibility with earlier tools, the per-component configurations
are loaded from `.rc` files in the working directory of each component.
The configuration names are also based on the original names, hence they
can be obscure. the `migrate` command allows for the conversion of the
`.rc` file to a JSON format one, the original `.rc` file being renamed
to end `.rc.orig` and allowing the `revert` command to restore the
original (without subsequent changes).

If you want to change settings you should first `migrate` the
configuration and then use `set` to make changes.

Note that execution mode (e.g. `GateMode`) is not supported and all
components run in the background.


#### Instance Configuration File

These configuration files - in JSON format -  should be found in
sub-directories under the `geneos` base directory (typiocally
`/opt/itrs`, `/opt/itrs/geneos` or `/opt/geneos`) as
`GENEOS_BASE_DIRECTORY/TYPE/TYPEs/INSTANCE/TYPE.json` where:

* `GENEOS_BASE_DIRECTORY` is the base directory for `geneos`.
* `TYPE` is the component type (`licd`, `gateway`, `netprobe`, `san`, `fa2`, `fileagent` or `webservcer`).
* `TYPEs` is the component type followed by the letter "s" (lowercase) to indicate a plural.
* `INSTANCE` is the instance name.
* `TYPE.json` is a the file name (e.g. `licd.json`, `gateway.json`, etc.).]

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




## Installation

### Download the binary

You can download a pre-built binary version (for Linux on amd64 only)
from [this
link](https://github.com/itrs-group/cordial/releases/latest/download/geneos)
or like this:

```bash
curl -OL https://github.com/itrs-group/cordial/releases/latest/download/geneos
chmod 555 geneos
sudo mv geneos /usr/local/bin/
```

### Build from source

To build from source you must have Go 1.20 or later installed:

#### One line installation

```bash
go install github.com/itrs-group/cordial/tools/geneos@latest
```

Make sure that the `geneos` program is in your normal `PATH` - or that
$HOME/go/bin is if you used the method above - to make things simpler.

#### Download from github and build manually

Make sure you do not have an existing file or directory called `geneos` and then:

```bash
github clone https://github.com/itrs-group/cordial.git
cd geneos/cmd/geneos
go build
sudo mv geneos /usr/local/bin
```
