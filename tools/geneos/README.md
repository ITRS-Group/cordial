# `geneos` management tool

The `geneos` program will help you manage your Geneos environment on Linux.

## Basic Features

* Initialise a new installation in one command
* Adopt an existing installation that uses older tools
* Manage a group of servers with a single command
* Manage certificates for TLS connectivity between Geneos components
* Configure the environment of components without editing files
* Download and install Geneos software, update components
* Simple bootstrapping of Self-Announcing Netprobes

## Aims

* Keep it simple through the [Principle of least astonishment](https://en.wikipedia.org/wiki/Principle_of_least_astonishment)
* Make your life easier; at least the part managing Geneos
* Help you use automation tools with Geneos

## Concepts & Terminology

Many of the terms used in this documentation and in the program itself assumes some familiarity with the Geneos suite of products and this is not always the case, so here are some starting points. Many of the key terms have been inherited from earlier systems.

The specific types supported by this program are details in [Component Types](#component-types) below.

### Geneos

[Geneos](https://www.itrsgroup.com/products/geneos) is a suite of software products from [ITRS](https://www.itrsgroup.com/) that provide real-time visibility of I.T. infrastructure and trading environments. It uses a three-tier architecture to collect, process and present enriched data to administrators.

### Components

A *component* is a type of software package and associated data. Each component will typically be a software package from one of the three-tiers mentioned above but can also be a derivative, e.g. a Self-Announcing Netprobe is a component type that abstracts the special configuration of either a vanilla Netprobe or, for example, the Fix Analyser Netprobe.

### Instances

An *instance* is an independent copy of a component with a working directory, configuration and other persistent files. Instances share read-only package directories for the binaries and other files from the distribution for the specific version being used.

### Hosts

*Hosts* are the locations that components are installed and instantiated. There is always a *localhost*.

## Getting Started

### Download the binary

You can download a pre-built binary version (for Linux on amd64 only) from [this link](https://github.com/itrs-group/cordial/releases/latest/download/geneos) or like this:

```bash
curl -OL https://github.com/itrs-group/cordial/releases/latest/download/geneos
chmod 555 geneos
sudo mv geneos /usr/local/bin/
```

### Build from source

To build from source you have Go 1.17+ installed:

#### One line installation

```bash
go install github.com/itrs-group/cordial/tools/geneos@latest
```

Make sure that the `geneos` program is in your normal `PATH` - or that $HOME/go/bin is if you used the method above - to make things simpler.

#### Download from github and build manually

Make sure you do not have an existing file or directory called `geneos` and then:

```bash
github clone https://github.com/itrs-group/cordial.git
cd geneos/cmd/geneos
go build
sudo mv geneos /usr/local/bin
```

### Adopting An Existing Installation

If you have an existing Geneos installation that you manage with the command like `gatewayctl`/`netprobectl`/etc. then you can use `geneos` to manage those once you have set the path to the Geneos installation.

| :warning: WARNING |
|:----------------------------|
| `geneos` ignores any changes to the global .rc files in your existing installation. You **must** check and adjust individual instance settings to duplicate settings. This can sometimes be very simple, for example if your `netprobectl.rc` files contains a line that sets `JAVA_HOME` then you can set this across all the Netprobes using `geneos set netprobe -e JAVA_HOME=/path/to/java`. More complex changes, such as library paths, will need careful consideration |

You can use the environment variable `ITRS_HOME` pointing to the top-level directory of your installation or set the location in the (user or global) configuration file:

```bash
geneos set user geneos=/path/to/install
```

This is the directory is where the `packages` and `gateway` (etc.) directories live. If you do not have an existing installation that follows this pattern then you can create a fresh layout further below.

Once you have set your directory you check your installation with some basic commands:

```bash
geneos ls     # list instances
geneos ps     # show their running status
geneos show   # show the default configuration values
```

None of these commands should have any side-effects but others will. These may not only start or stop processes but may also convert configuration files to JSON format without prompting. Old `.rc` files are backed-up with a `.rc.orig` extension and can be restored using the `revert` command.

### New Installation

New installations are set-up through the `init` sub-command. In it's most basic form it will create the minimal directory hierarchy and your user-specific geneos.json file containing the path to the top-level directory that it initialised. The top-level directory, if not given on the command line, defaults to a directory `geneos` in your home directory *unless* the last part of your home directory is itself `geneos`, e.g. if your home directory is `/home/example` then the Geneos directory becomes `/home/example/geneos` but if it is `/opt/geneos` then that is used directly.

If the directory you are using is not empty then you must supply a `-F` flag for force using this directory.

#### Demo Gateway

You can set-up a Demo environment like this:

```bash
geneos init -D
```

If authentication is required to download the software archives then use these extra options:

```bash
geneos init -D -u user@example.com
```

or

```bash
export ITRS_DOWNLOAD_USERNAME=user@example.com
export ITRS_DOWNLOAD_PASSWORD=mysecret
geneos init -D
```

Here you should replace the email address with your own and the command will prompt you for your password. These are the login details you should have for the ITRS Resources website.

The above command will create a directory structure, download software and configure a Gateway in 'Demo' mode plus a single Self-Announcing Netprobe and Webserver for dashboards. However, no further configuration is done, that's up to you!

Behind the scenes the command does roughly this for you:

```bash
geneos init
geneos install gateway -u ...
geneos add gateway 'Demo Gateway'
geneos install san -u ...
geneos add san localhost -g localhost
geneos install webserver -u ...
geneos add webserver demo
geneos start
geneos ps
```

#### Self-Announcing Netprobe

You can install a Self-Announcing Netprobe (SAN) in one line, like this:

```bash
geneos init -S -n SAN123 -c /path/to/signingcertkey \
    -g gateway1 -g gateway2 -t Infrastructure -t App1 -t App2 \
    -a ENVIRONMENT=Prod -a LOCATION=London -u user@example.com
```

This example will create a SAN with the name SAN123 connecting, using TLS, to gateway1 and gateway2, using types and attributes as listed.

Again, you can add authentication options for the downloads using `-u` and `-p`.

#### Another Initial Environment

```bash
geneos init -A geneos.lic -u user@example.com
```

does this (where HOSTNAME is, of course, replaced with the hostname of the server)

```bash
geneos init
geneos install gateway -u ...
geneos new gateway HOSTNAME
geneos install san -u ...
geneos new san HOSTNAME -g localhost
geneos install licd -u ...
geneos new licd HOSTNAME
geneos install webserver -u ...
geneos new webserver HOSTNAME
geneos import licd HOSTNAME geneos.lic
geneos start
```

Instance names are case sensitive and cannot be the same as some reserved words (e.g. `gateway`, `netprobe`, `probe` and more, given below).

You still have to configure the Gateway to connect to the Netprobe, but all three components should now be running. You can check with:

```bash
geneos ps
```

## Security and Running as Root

This program has been written in such a way that is *should* be safe to install SETUID root or run using `sudo` for almost all cases. The program will refuse to accidentally run an instance as root unless the `User` config parameter is explicitly set - for example when a Netprobe needs to run as root. As with many complex programs, care should be taken and privileged execution should be used when required.

It is worth reminding users that environment variables do not get passed to programs run with `sudo` unless you use the `-E` option (and you are permitted to use it). This is especially important where you have set the download credentials in environment variables. For example, this is likely to fail:

```bash
export ITRS_DOWNLOAD_USERNAME=email@example.com
export ITRS_DOWNLOAD_PASSWORD=supersecret

sudo -u geneos geneos install gateway
```

While adding the -E flags like this will work:

```bash
sudo -E -u geneos geneos install gateway
```

## Environment Settings

The `geneos` program uses the packages [Cobra](cobra.dev) and [Viper](https://github.com/spf13/viper) (the latter via a wrapper package) to provide the command syntax and configuration management. There is full support for Viper's layered configuration for non-instance settings, which means you can override global and user settings with environment variables prefixed `ITRS_`, e.g. `ITRS_DOWNLOAD_USERNAME` overrides `download.username`

## Instance Settings

Each instance has a configuration file. This is the most basic expression of an instance. New instances that you create will have a configuration file named after the component type plus the extension `.json`. Older instances which you have adopted from older control scripts will have a configuration file with the extension `.rc`

### Historical Configuration Files

Historical (legacy) `.rc` files have lines, ignoring comments, of form

```bash
GatePort=1234
```

Where the prefix (`Gate`) also encodes the component type and the suffix (`Port`) is the setting. Any lines that do not contain the prefix are treated as environment variables and are evaluated and passed to the program on start.

While the `geneos` program can parse and understand the legacy `.rc` files above it will never update them, instead migrating them to their modern `.json` versions either when required or when explicitly told to using the `migrate` command.

### JSON Configuration Files

The `.json` configuration files share common parameters as well as component type specific settings. For brevity some of these parameters are overloaded and have different meanings depending on the component type they apply to.

While editing the configuration files directly is possible, it is best to use the `set` and `unset` commands to ensure the syntax is correct.

### Special parameters

All instance types support custom environment variables being set or unset. This is done through the `set` and `unset` commands below, alongside the standard configuration parameters for each instance type.

Some component types, namely Gateways and SANs, support other special parameters via other command line flags. See the help text or the full documentation for the `set` and `unset` commands for more details.

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

Finally, if your environment variable value contains spaces then use quotes as appropriate to your shell to prevent those spaces being processed. In bash you can do any of these to achieve the same result:

```bash
geneos set netprobe example1 -e MYVAR="a string with spaces"
geneos set netprobe example1 -e "MYVAR=a string with spaces"
```

You can review the environment for any instance using the `show` command:

```bash
geneos show netprobe example1
```

Also. output is available from the `command` command to show what would be run when calling the `start` command:

```bash
geneos command netprobe example1
```

## Component Types

The following component types (and their aliases) are supported:

* **`gateway`** - or `gateways`

* **`netprobe`** - or `netprobes`, `probe` or `probes`

* **`licd`** - or `licds`

* **`webserver`** - or `webservers`, `webdashboard`. `dashboards`

* **`san`** - or `sans`

* **`fa2`** - or `fixanalyser`, `fix-analyser`

* **`fileagent`** - or `fileagents`

* `any` (which is the default)

The first name, in bold, is also the directory name used for each type. These names are also reserved words and you cannot configure (or consistently manage) components with those names. This means that you cannot have a gateway called `gateway` or a probe called `probe`. If you do already have instances with these names then you will have to be careful migrating. See more below.

Each component type is described below along with specific component options.

### Type `gateway`

* Gateway general

* Gateway templates

  When creating a new Gateway instance a default `gateway.setup.xml` file is created from the template(s) installed in the `gateway/templates` directory. By default this file is only created once but can be re-created using the `rebuild` command with the `-F` option if required. In turn this can also be protected against by setting the Gateway configuration setting `configrebuild` to `never`.

* Gateway variables for templates

  Gateways support the setting of Include files for use in templated configurations. These are set similarly to the `-e` parameters:

  ```bash
  geneos gateway set example2 -i  100:/path/to/include
  ```

  The setting value is `priority:path` and path can be a relative or absolute path or a URL. In the case of a URL the source is NOT downloaded but instead the URL is written as-is in the template output.

### Type `netprobe`

* Netprobe general

### Type `licd`

* Licd general

### Type `webserver`

* Webserver general

* Java considerations

* Configuration templates - TBD

### Type `san`

* San general

* San templates

* San variables for templates

  Like for Gateways, SANs get a default configuration file when they are created. By default this is from the template(s) in `san/templates`. Unlike for the Gateway these configuration files are rebuilt by the `rebuild` command by default. This allows the administrator to maintain SANs using only command line tools and avoid having to edit XML directly. Setting `configrebuild` to `never` in the instance configuration prevents this rebuild.
  To aid this, SANs support the following special parameters:

  * Attributes

  Attributes can be added via `set`, `add` or `init` using the `-a` flag in the form NAME=VALUE and also removed using `unset` in the same way but just with a NAME

  * Gateways

  As for Attributes, the `-g` flag can specify Gateways to connect to in the form HOSTNAME:PORT

  * Types

  Types can be specified using `-t`

  * Variables

  Variables can be set using `-v` but there is only support for a limited number of types, specifically those that have values that can be give in plain string format.

* Selecting the underlying Netprobe type (For Fix Analyser 2 below)
  A San instance will normally be built to use the general purpose Netprobe package. To use an alternative package, such as the Fix Analyser 2 Netprobe, add the instance with the special format name `fa2:example[@REMOTE]` - this configures the instance to use the `fa2` as the underlying package. Any future special purpose Netprobes can also be supported in this way.

### Type `fa2`

* Fix Analyser 2 general

### Type `fileagent`

* File Agent general

## Remote Management

The `geneos` command can transparently manage instances across multiple systems using SSH.

### What does this mean?

See if these commands give you a hint:

```bash
geneos add host server2 ssh://geneos@myotherserver.example.com/opt/geneos
geneos add gateway newgateway@server2
geneos start
```

Command like `ls` and `ps` will works transparently and merge all instances together, showing you where they are configured to run.

The format of the SSH URL has been extended to include the Geneos directory and for the `add host` command is:

`ssh://[USER@]HOST[:PORT][/PATH]`

If not set, USER defaults to the current username. Similarly PORT defaults to 22. PATH defaults to the local Geneos path. The most basic SSH URL of the form `ssh://hostname` results in a remote accessed as the current user on the default SSH port and rooted in the same directory as the local set-up. Is the remote directory is empty (dot files are ignored) then the standard file layout is created. If you do not provide any SSH URL then the hostname is taken from the name of the host - e.g.

```bash
geneos add host myserver
```

is taken as:

```bash
geneos add host myserver ssh://myserver
```

### How does it work?

There are a number of prerequisites for remote support:

1. Remote hosts must be Linux on amd64

2. Password-less SSH access, either via an `ssh-agent` or unprotected private keys

3. At this time the only private keys supported are those in your `.ssh` directory beginning `id_` - later updates will allow you to set the name of the key to load, but using an agent is recommended.

4. The remote user must be configured to use a `bash` shell or similar. See limitations below.

If you can log in to a remote Linux server using `ssh user@server` and not be prompted for a password or passphrase then you are set to go. It's beyond the scope of this README to explain how to set-up `ssh-agent` or how to create an unprotected private key file, so please search online.

### Limitations

The remote connections over SSH mean there are limitations to the features available on remote servers:

1. Control over instance processes is done via shell commands and little error checking is done, so it is possible to cause damage and/or processes not to to start or stop as expected. Contributions of fixes are welcomed.

2. All actions are taken as the user given in the SSH URL (which should NEVER be `root`) and so instances that are meant to run as other users cannot be controlled. Files and directories may not be available if the user does not have suitable permissions.

## Usage

CAUTION: Please note that the full list of commands and parameters is still changing at this time. This list below is mostly, but not completely, up-to-date.

The general syntax is:

`geneos COMMAND [FLAGS] [TYPE] [NAMES...]`

There are a number of special cases, these are detailed below.

### Commands

The following commands are available:

```txt
Available Commands:
  add         Add a new instance
  aes         Manage Gateway AES key files
  clean       Clean-up instance directories
  command     Show command line and environment for launching instances
  completion  Generate the autocompletion script for the specified shell
  copy        Copy instances
  delete      Delete an instance. Instance must be stopped
  disable     Stop and disable instances
  enable      Enable instances. Only previously disabled instances are started
  help        Help about any command
  home        Print the home directory of the first instance or the Geneos home dir
  import      Import file(s) to an instance or a common directory
  init        Initialise a Geneos installation
  install     Install files from downloaded Geneos packages. Intended for sites without Internet access
  logs        Show log(s) for instances
  ls          List instances, optionally in CSV or JSON format
  migrate     Migrate legacy .rc configuration to new .json format
  move        Move (or rename) instances
  ps          List process information for instances, optionally in CSV or JSON format
  rebuild     Rebuild instance configuration files
  reload      Reload instance configuration, where supported
  restart     Restart instances
  revert      Revert migration of .rc files from backups
  set         Set instance configuration parameters
  show        Show runtime, global, user or instance configuration is JSON format
  snapshot    Capture a snapshot of each matching dataview
  start       Start instances
  stop        Stop instances
  tls         Manage certificates for secure connections
  unset       Unset a configuration value
  update      Update the active version of Geneos software
```

#### General Command Flags & Arguments

`geneos [FLAG...] COMMAND [FLAG...] [TYPE] [NAME...] [PARAM...]`

Where:

* `FLAG` - Both general and command specific flags
* `COMMAND` - one of the configured commands
* `TYPE` - the component type
* `NAME` - one or more instance names, optionally including the remote server
* `PARAM` - anything that isn't one of the above

In general, with the exception of `COMMAND` and `TYPE`, all parameters can be in any order as they are filtered into their types for most commands. Some commands require arguments in an exact order. For example, these have the same meaning:

`geneos ls -c gateway one two three`
`geneos ls gateway one -c two three`

Reserved instance names are case-insensitive. So, for example, "gateway", "Gateway" and "GATEWAY" are all reserved.

The `NAME` is of the format `INSTANCE@REMOTE` where either is optional. In general commands will wildcard the part not provided. There are special `REMOTE` names `@localhost` and `@all` - the former is, as the name suggests, the local server and `@all` is the same as not providing a remote name.

There is a special format for adding SANs in the form `TYPE:NAME@REMOTE` where `TYPE` can be used to select the underlying Netprobe type. This format is still accepted for all other commands but the `TYPE` is silently ignored.

#### File and URLs

In general all source file references support URLs, e.g. importing certificate and keys, license files, etc.

The primary exception is for Gateway include files used in templated configurations. If these are given as URLs then they are used in the configuration as URLs.

#### Global Commands

* `geneos version`

Show the current version of the `geneos` program, which should match the tag of the overall `geneos` package.

* `geneos help`

General help, initially a list of all the supported commands.

* `geneos completion` - **Not yet implemented**

Output completion configuration information for most shells.

#### Instance Details

* `geneos ls [TYPE] [NAME...]`

Output a list of all configured instances. If a TYPE and/or NAME(s) are supplied then list those that match.

* `geneos ps [TYPE] [NAME...]`

Show details of running instances. This includes, where available, which TCP ports a process is listening on.

* `geneos logs [-f | -n N | ...] [TYPE] [NAME...]`

Show log(s) for matching instances. Flags allow for follow etc.

#### Environment Commands

* `geneos init [FLAGS] [USERNAME] [PATH] [PARAMS]`

  The `init` command initialises a Geneos installation. It has many - perhaps too many - options. It is very likely that this command will change in the future to support multiple sub-commands to make the options more manageable.

  The FLAGS are:

  * `-A LICENSE` Initialise and start an `all` environment
    Initialise the environment. This command creates a directory hierarchy and optionally installs Geneos software packages and also optionally creates instances and starts them.

  * `-D` Initialise and start a demo environment

  * `-S` Initialise and start a San. See the `-n` option below. Takes all the same PARAMS as for adding a San to specify template settings.

  * `-C` Create a root and signing certificate and use these for all new instances

  * `-l` Follow logs after starting any created instances (for `-A`, `-D` and `-S` above)

  * `-F` Force initialisation, ignore existing files and directories in destination

  * `-n NAME` Use the `NAME` for instances instead of the default hostname. This is especially useful for Sans and Gateways as the templates use this name to fill in various configuration item defaults

  * `-c CERT` and/or `-k KEY` Import certificates and keys during initialisation. See `geneos tls import` for more details. When a valid signing certificate and key are imported then all subsequent new instances will have individual certificates and keys created.

  * `-T` Rebuild the default templates using the embedded files. This is primarily to update templates when new versions of this program are release or if they have become corrupted

  * `-N` Download from `nexus.itrsgroup.com` for testing. Only available to ITRS staff.

  * `-p` Use the Nexus `snapshots` repository instead of the default `releases` when using `-N` above

  * `-V` Version filter for downloads. Defaults to the latest version found.

  * `-w FILE` A Gateway template file to use instead of the embedded one

  * `-s FILE` A San template file to use instead of the embedded one

  * `-e`, `-i`, `-g`, `-a`, `-t`, `-v` These options set environment, include, gateway. attribute, type and variable values and can be specified multiple times. The values are used as inputs to templates for Gateway and SAN instances.

  * Only one of the `-T`, `-S`, `-D` or `-A` options are valid and only the `-T` option can be used for multiple calls to this command unless `-F` is also used.

* `geneos tls`

  TLS operations. See below.

* `geneos show [global|user]`

  Show the running configuration or, if `global` or `user` is supplied then the respective on-disk configuration files. Passwords are simplistically redacted.

  The instance specific `show` command is described below.

* `geneos set [global|user] KEY=VALUE...`

  Set a program-wide configuration option. The default is to update the `user` configuration file. If `global` is given then the user has to have appropriate privileges to write to the global config file (`/etc/geneos/geneos.json`). Multiple KEY=VALUE pairs can be given but only fields that are recognised are updated.

* `geneos home [TYPE] [NAME]`

  The `home` command outputs the home directory of the first matching instance, or `Geneos` if there is no match or no options passed to the command. This is useful for automation and shortcuts, e.g. in bash:

  ```bash
  cd $(geneos home netprobe example1)
  ```

Please note that if `geneos home` returns an empty string because of an error the cd command will take you to your home directory.

#### Package Management Commands

* `geneos install [FLAGS] [TYPE] [latest|VERSION|FILE|URL...]`

  Download and install a release archive in the `packages` directory. If a symbolic link for the desired base version already exists it is not updated unless the `-U` flag is given.

  The FLAGS are:

  * `-b BASENAME` The base name of the installation symbolic link. Defaults to `active_prod`

  * `-l` Local archives only

  * `-n` No save of downloaded archives

  * `-H HOST` Install only on remote HOST. HOST must be configured

  * `-N` Use `nexus.itrsgroup.com` for downloads. Only available to ITRS staff

  * `-p` Use the `snapshots` repository and not the default `releases` with `-N` above

  * `-V` Version filter, installs latest matching version. Defaults to latest available version.

  * `-U` Update base name link to installed version

  * `-T TYPE:VERSION` Override the component type and version numbers to allow installation from files with generic names, for example `gateway.tgz`

* `geneos update [TYPE] [VERSION]`

  Update the component base binary symlink

#### Control Commands

* `geneos start [-l] [TYPE] [NAME...]`

  Start a Geneos component. If no name is supplied or the special name `all` is given then all the matching Geneos components are started.

* `geneos stop [-K] [TYPE] [NAME...]`

  Like above, but stops the component(s)

  * `-K` terminates forcefully - i.e. a SIGKILL is immediately sent

* `geneos restart [-l] [TYPE] [NAME...]`

  Restarts matching geneos components. Each component is stopped and started in sequence. If all components should be down before starting up again then use a combination of `start` and `stop` from above.

* `geneos reload [TYPE] NAME [NAME...]`

  Signal the component to reload it's configuration or restart as appropriate.

* `geneos disable [TYPE] [NAME...]`

  Stop and disable the selected components by placing a file in the working directory with a `.disable` extension

* `geneos enable [TYPE] [NAME...]`

  Remove the `.disable` lock file and start the selected components

* `geneos clean [-F] [TYPE] [names]`

  Clean up component directory. Optionally 'full' clean, with an instance restart.

#### Configuration Commands

* `geneos add [TYPE] NAME [NAME...]`

  Add a new Geneos component configuration.

* `geneos migrate [TYPE] [NAME...]`

  Migrate legacy `.rc` files to `.json` and backup the original file with an `.orig` extension. This backup file can be used by the `revert` command, below, to restore the original `.rc` file(s)

* `geneos revert [TYPE] [NAME...]`

  Revert to the original configuration files, deleting the `.json` files. Note that the `.rc` files are never changed and any configuration changes to the `.json` configuration will not be retained.

* `geneos rebuild [-n] [-F] [TYPE] [NAME...]`

  Rebuild instance configuration, typically used for Self-Announcing Netprobes. By default it restarts any instances where the configuration has changed. Flags are:

  * `-n` Do not restart instances
  * `-f` Force rebuild for those instances that are marked `initial` only.

* `geneos command [TYPE] [NAME...]`

  Shows details of the full command used for the component and any extra environment variables found in the configuration.

* `geneos move|copy [TYPE] source destination`

  These commands move or copy instance(s), respectively. If the source and destination are on the same location/remote then `move` acts as a rename. If a destination is given as a bare remote, e.g. `@remotename` then the source instance name is kept. If both source and destination are remotes then all matching instances are moved or copied. If no TYPE is given then all matching instances will be acted on. If the destination is a different location/remote then the port number of the instance(s) may be updated to avoid clashing with existing instances. Because all changes require the writing of a new instance configuration file, all instances are migrated to new JSON configuration files if required.

* `geneos delete [-F] component name`

  Deletes the disabled component given. Only works on components that have been disabled beforehand, unless the `-F` (force) flag is supplied.

* `geneos import [-c common] [-r host] [TYPE] [NAME] [file|url|-]...`

  Import a file into an instance working directory, from local file, url or stdin and backup previous file. The file can also specify the destination name and sub-directory, which will be created if it does not exist. Examples of valid files are:

  ```bash
  geneos import gateway Example gateway.setup.xml
  geneos import gateway Example https://server/files/gateway.setup.xml
  geneos import gateway Example gateway.setup.xml=mygateway.setup.xml
  geneos import gateway Example scripts/newscript.sh=myscript.sh
  geneos import gateway Example scripts/=myscript.sh
  cat someoutput | geneos import gateway Example config.json=-
  ```

  Like other commands that write to the file system, it can safely be run as root as the destination directory and file will be changed to be owned by either the instance or the default user, with the caveat that any intermediate directories created above the destination directory (e.g. the first two in `my/long/path`) will be owned by root.

* `geneos snapshot [FLAGS] [TYPE] [NAME} XPATH [XPATH...]`

  Snapshot one or more dataviews using the REST Commands API endpoint introduced in GA5.14. The TYPE, if given, must be `gateway`. Authentication to the Gateway is through a combination of command line flags and configuration parameters. If either of the parameters `snapshot.username` or `snapshot.password` is defined for the Gateway or globally then this is used as a default unless overridden on the command line by the `-u` and `-P` options. The user is only prompted for a password if it cannot be located in either of the previous places.

  The output is in JSON format as an array of dataviews, where each dataview is in the format defined in the [Gateway documentation](https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_commands_tr.html#fetch_dataviews). Flags to select which properties of data items are available: `-V`, `-S`, `-Z`, `-U` for value, severity, snooze and user-assignment respectively. If none is given then the default is to fetch values only.

  To help capture diagnostic information the `-x` option can be used to capture matching xpaths without the dataview contents. `-l` can be used to limit the number of dataviews (or xpaths) but the limit is not applied in any defined order.

## Secure Passwords

The `geneos aes` commands provide tools to manage Geneos AES256 key files as [documented here](https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm)

In addition to the functionality built-in to Geneos as described in the Gateway documentation these encoded password can also be included in configuration files so that plain text passwords and other credentials are not visible to users.

* `geneos aes ls`

  List configured key-files in Geneos components. The CRC32 column is provided as a visual aid to human users to identify common key-files.
  
  Note: If a key-file is configured then the component - currently only Gateways - are started with the key-file on the command line. This may cause start-up issues if the key-file has just been added or changed and your Gateway is earlier than GA5.14.0 or there is an existing `cache/` directory in the Gateway working directory. To resolve this you may have to remove the `cache/` directory (use the `geneos clean` command with the `-F` full-clean option) or start the Gateway with a `-skip-cache` option which can be set with `geneos set -k options=-skip-cache` and so on.

* `geneos aes encode [-k KEYFILE] [-p STRING] [-s SOURCE] [TYPE] [NAME]`

  Encode a plain text password using the key-file given or the key-files configured for any matching instances. If instances share the same key-file then the same output will be generated for each. If neither a string or a source path is given then the user is prompted to enter a password. The SOURCE can be a local file or a URL.

* `geneos aes decode [-k KEYFILE] [-v KEYFILE] [-p PASSWORD] [-s SOURCE] [TYPE] [NAME]`

  Decode the encoded text using the given key-file or previous key-file, if given on the command line or using the key-files for matching instances. The first valid UTF-8 decoded text is output and further processing stops. The encoded text can be prefixed with the Geneos `+encs+` text, which will be removed if present. The SOURCE can be a local file or a URL.

* `geneos aes new -k PATH -S [TYPE] [NAME]`

  Create a new key-file. With no arguments a new file is created in the current directory called `keyfile.aes`. If the set option (`-S`) is provided then the keyfile is copied to the `gateway/gateway_shared/keyfiles` directory with a (hopefully) unique name (the CRC32 of the file plus `.aes`), synced to remote hosts and all matching instances have their key-file parameters set to use this file. Unlike `geneos aes sync` below, only the new key-file is copied to the shared location.

* `geneos aes update` - **Not yet implemented**

  Update the existing key-file in use by rotating the currently configured key-file to previous-key-file. Requires GA6.x.

* `geneos aes sync` - **Not yet implemented**

  Sync local key-file(s) to remote hosts.

## TLS Operations

The `geneos tls` command provides a number of subcommands to create and manage certificates and instance configurations for encrypted connections.

Once enabled then all new instances will also have certificates created and configuration set to use secure (encrypted) connections where possible.

The root and signing certificates are only kept on the local server and the `tls sync` command can be used to copy a `chain.pem` file to remote servers. Keys are never copied to remote servers by any built-in commands.

* `geneos tls init`

  Initialised the TLS environment by creating a `tls` directory in Geneos and populating it with a new root and intermediate (signing) certificate and keys as well as a `chain.pem` which includes both CA certificates. The keys are only readable by the user running the command. Also does a `sync` if remotes are configured.

  Any existing instances have certificates created and their configurations updated to reference them. This means that any legacy `.rc` configurations will be migrated to `.json` files.

* `geneos tls import FILE [FILE...]`

  Import certificates and keys as specified to the `tls` directory as root or signing certificates and keys. If both certificate and key are in the same file then they are split into a certificate and key and the key file is permissioned so that it is only accessible to the user running the command.

  Root certificates are identified by the Subject being the same as the Issuer, everything else is treated as a signing key. If multiple certificates of the same type are imported then only the last one is saved. Keys are checked against certificates using the Public Key part of both and only complete pairs are saved.

* `geneos tls new [TYPE] [NAME...]`

  Create a new certificate for matching instances, signed using the signing certificate and key. This will NOT overwrite an existing certificate and will re-use the private key if it exists. The default validity period is one year. This cannot currently be changed.

* `geneos tls renew [TYPE] [NAME...]`

  Renew a certificate for matching instances. This will overwrite an existing certificate regardless of it's current status of validity period. Any existing private key will be re-used. `renew` can be used after `import` to create certificates for all instances, but if you already have specific instance certificates in place you should use `new` above.
  As for `new` the validity period is a year and cannot be changed at this time.

* `geneos tls ls [-a] [-c|-j] [-i] [-l] [TYPE] [NAME...]`

  List instance certificate information. Flags are similar as for the main `ls` command but the data shown is specific to certificates. Additional flags are:

  * `-a` List all certificates. By default the root and signing certificates are not shown
  * `-l` Long list format, which includes the Subject and Signature. This signature can be used directly in the Geneos Authentication entry for users for non-user authentication using client certificates, e.g. Gateway Sharing and Web Server.

* `geneos tls sync`

  Copies chain.pem to all remotes

## Configuration Files

### General Configuration

* `/etc/geneos/geneos.json` - Global options
* `${HOME}/.config/geneos.json` - User options
* Environment variables ITRS_option

General options are loaded from the global config file first, then the user one and any environment variables override both files. The current options are:

* `geneos`
The home directory for all other commands. See [Directory Layout](#directory-layout) below. If set the environment variable ITRS_HOME overrides any settings in the files. This is to maintain backward compatibility with older tools. The default, if not set anywhere else, is the home directory of the user running the command or, if running as root, the home directory of the `geneos` or `itrs` users (in that order). (To be fully implemented)
This value is also set by the environment variables `ITRS_HOME` or `ITRS_GENEOS`

* `download.url`
The base URL for downloads for automating installations. Not yet used.
If files are locally downloaded then this can either be a `file://` style URL or a directory path.

* `download.username`
  `download.password`
  These specify the username and password to use when downloading packages. They can also be set as the environment variables, but the environment variables are not subject to expansion and so cannot contain Geneos encoded passwords (see below):
  * `ITRS_DOWNLOAD_USERNAME`
  * `ITRS_DOWNLOAD_PASSWORD`

* `snapshot.username`
  `snapshot.password`
  Similarly to the above, these specify the username and password to use when taking dataview snapshots. They can also be set as the environment variables, with the same restrictions as above:
  * `ITRS_SNAPSHOT_USERNAME`
  * `ITRS_SNAPSHOT_PASSWORD`

* `defaultuser`
Principally used when running with elevated privilege (setuid or `sudo`) and a suitable username is not defined in instance configurations or for file ownership of shared directories.

* `GatewayPortRange` & `NetprobePortRange` & `LicdPortRange`
...

### Component Configuration

For compatibility with earlier tools, the per-component configurations are loaded from `.rc` files in the working directory of each component. The configuration names are also based on the original names, hence they can be obscure. the `migrate` command allows for the conversion of the `.rc` file to a JSON format one, the original `.rc` file being renamed to end `.rc.orig` and allowing the `revert` command to restore the original (without subsequent changes).

If you want to change settings you should first `migrate` the configuration and then use `set` to make changes.

Note that execution mode (e.g. `GateMode`) is not supported and all components run in the background.

## Directory Layout

The `geneos` configuration setting or the environment variable `ITRS_HOME` points to the base directory for all subsequent operations. The layout follows that of the original `gatewayctl` etc. including:

```text
packages/
  gateway/
    [versions]/
    active_prod -> [chosen version]
  netprobe/
  licd/
gateway/
netprobe/
licd/
```

The `bin/` directory and the default `.rc` files are **ignored** so be aware if you have customised anything in `bin/`.

As a very quick recap, each component directory will have a subdirectory with the plural of the name (e.g. `gateway/gateways`) which will contain subdirectories, one per instance, and these act as the configuration and working directories for the individual processes. Taking an example gateway called `Gateway1` the path will be:

`${ITRS_HOME}/gateway/gateways/Gateway1`

This directory will be the working directory of the process and also contain an `.rc` configuration file as well as a `.txt` file to capture the `STDOUT` and `STDERR` of the process, like this:

```bash
gateway.rc
gateway.txt
```

There will also be an XML setup file and so on.
