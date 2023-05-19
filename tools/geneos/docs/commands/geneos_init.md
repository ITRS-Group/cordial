# `geneos init`

Initialise a Geneos installation

```text
geneos init [flags] [USERNAME] [DIRECTORY]
```
## Commands

* [`geneos init all`](geneos_init_all.md)	 - Initialise a more complete Geneos environment
* [`geneos init demo`](geneos_init_demo.md)	 - Initialise a Geneos Demo environment
* [`geneos init floating`](geneos_init_floating.md)	 - Initialise a Geneos Floating Netprobe environment
* [`geneos init san`](geneos_init_san.md)	 - Initialise a Geneos SAN (Self-Announcing Netprobe) environment
* [`geneos init template`](geneos_init_template.md)	 - Initialise or overwrite templates

## Details

# `geneos init` Subsystem Commands


Initialise a Geneos installation by creating the directory
structure and user configuration file, with the optional username and directory.

- `USERNAME` refers to the Linux username under which the `geneos` utility
  and all Geneos component instances will be run.
- `DIRECTORY` refers to the base / home directory under which all Geneos
  binaries, instances and working directories will be hosted.
  When specified in the `geneos init` command, DIRECTORY:
  - Must be defined as an absolute path.
    This syntax is used to distinguish it from USERNAME which is an
    optional parameter.
	If undefined, `${HOME}/geneos` will be used, or `${HOME}` in case
	the last component of `${HOME}` is equal to `geneos`.
  - Must have a parent directory that is writeable by the user running 
    the `geneos init` command or by the specified USERNAME.
  - Must be a non-existing directory or an empty directory (except for
	the "dot" files).
	**Note**:  In case DIRECTORY is an existing directory, you can use option
	`-F` to force the use of this directory.

The generic command syntax is as follows.
` geneos init [flags] [USERNAME] [DIRECTORY] `

When run with superuser privileges a USERNAME must be supplied and
only the configuration file for that user is created.
` sudo geneos init geneos /opt/itrs `

**Note**:
- The geneos directory hierarchy / structure / layout is defined at
  [Directory Layout](https://github.com/ITRS-Group/cordial/tree/main/tools/geneos#directory-layout).

## Adopting An Existing Installation

If you have an existing Geneos installation that you manage with the
command like `gatewayctl`/`netprobectl`/etc. then you can use `geneos`
to manage those once you have set the path to the Geneos installation.

| :warning: WARNING |
|:----------------------------|
| `geneos` ignores any changes to the global `.rc` files in your |
| existing installation. You **must** check and adjust individual instance |
| settings to duplicate settings. This can sometimes be very simple, for |
| example if your `netprobectl.rc` files contains a line that sets |
| `JAVA_HOME` then you can set this across all the Netprobes using `geneos |
| set netprobe -e JAVA_HOME=/path/to/java`. More complex changes, such as |
| library paths, will need careful consideration |

You can use the environment variable `ITRS_HOME` pointing to the
top-level directory of your installation or set the location in the
(user or global) configuration file:

```bash
geneos set user geneos=/path/to/install
```

This is the directory is where the `packages` and `gateway` (etc.)
directories live. If you do not have an existing installation that
follows this pattern then you can create a fresh layout further below.

Once you have set your directory you check your installation with some
basic commands:

```bash
geneos ls     # list instances
geneos ps     # show their running status
geneos show   # show the default configuration values
```

None of these commands should have any side-effects but others will.
These may not only start or stop processes but may also convert
configuration files to JSON format without prompting. Old `.rc` files
are backed-up with a `.rc.orig` extension and can be restored using the
`revert` command.

## New Installation

New installations are set-up through the `init` sub-command. In it's
most basic form it will create the minimal directory hierarchy and your
user-specific `geneos.json` file containing the path to the top-level
directory that it initialised. The top-level directory, if not given on
the command line, defaults to a directory `geneos` in your home
directory *unless* the last part of your home directory is itself
`geneos`, e.g. if your home directory is `/home/example` then the Geneos
directory becomes `/home/example/geneos` but if it is `/opt/geneos` then
that is used directly.

If the directory you are using is not empty then you must supply a `-F`
flag to  force the use of this directory.

### Demo Gateway

You can set-up a Demo environment like this:

```bash
geneos init demo -i email@example.com
```

or, to script this, do:

```bash
export ITRS_DOWNLOAD_USERNAME=email@example.com
export ITRS_DOWNLOAD_PASSWORD=mysecret
geneos init demo
```

Here you should replace the email address with your own and the command
will prompt you for your password. These are the login details you
should have for the [ITRS Resources
website](https://resources.itrsgroup.com/).

The above command will create a directory structure, download software
and configure a Gateway in 'Demo' mode plus a single Self-Announcing
Netprobe and Webserver for dashboards. However, no further configuration
is done, that's up to you!

Behind the scenes the command does (approximately) this for you:

```bash
geneos init
geneos install gateway -u ...
geneos add gateway 'Demo Gateway'
geneos install san -u ...
geneos add netprobe localhost -g localhost
geneos install webserver -u ...
geneos add webserver demo
geneos start
geneos ps
```

### Self-Announcing Netprobe

You can install a Self-Announcing Netprobe (SAN) in one line, like this:

```bash
geneos init san -n SAN123 -c /path/to/signingcertkey \
    -g gateway1 -g gateway2 -t Infrastructure -t App1 -t App2 \
    -a ENVIRONMENT=Prod -a LOCATION=London -u email@example.com
```

This example will create a SAN with the name SAN123 connecting, using
TLS, to gateway1 and gateway2, using types and attributes as listed.

### A More Complete Initial Environment

```bash
geneos init all -L ./geneos.lic -u email@example.com
```

does this (where HOSTNAME is, of course, replaced with the hostname of
the server)

```bash
geneos init
geneos install gateway -u ...
geneos add gateway HOSTNAME
geneos install san -u ...
geneos add netprobe HOSTNAME -g localhost
geneos install licd -u ...
geneos add licd localhost
geneos install webserver -u ...
geneos add webserver HOSTNAME
geneos import licd HOSTNAME geneos.lic
geneos start
```

Instance names are case sensitive and cannot be the same as some
reserved words (e.g. `gateway`, `netprobe`, `probe` and more, given
below).

You still have to configure the Gateway to connect to the Netprobe, but
all three components should now be running. You can check with:

```bash
geneos ps
```

### Options

```text
  -l, --log                       Follow logs after starting instance(s)
  -F, --force                     Be forceful, ignore existing directories.
  -n, --name string               Use name for instances and configurations instead of the hostname
  -C, --makecerts                 Create default certificates for TLS support
  -c, --importcert string         signing certificate file with optional embedded private key
  -k, --importkey string          signing private key file
  -N, --nexus                     Download from nexus.itrsgroup.com. Requires ITRS internal credentials
  -p, --snapshots                 Download from nexus snapshots. Requires -N
  -V, --version string            Download matching version, defaults to latest. Doesn't work for EL8 archives. (default "latest")
  -u, --username string           Username for downloads
  -w, --gatewaytemplate string    A gateway template file
  -s, --santemplate string        SAN template file
  -f, --floatingtemplate string   Floating probe template file
  -e, --env NAME=VALUE            An environment variable for instance start-up
                                  (Repeat as required)
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## Examples

```bash
# To create a Geneos tree under home area
geneos init
# To create a new Geneos tree owned by user `geneos` under `/opt/itrs`
sudo geneos init geneos /opt/itrs

```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
