# `geneos init` Subsystem Commands

The `init` commands initialise your Geneos install environment ready to
run Geneos instances.

On it's own the `init` command will create a new directory (or use an
existing one if it's considered empty) based on the options used. By
default it will create a directory named `geneos` in your user's home
directory unless your home directory ends in `geneos` (e.g.
`/home/geneos`) in which case it tries that to avoid stuttering in the
path.

The installation directory is considered empty if it exists but only
contains `dot` files and directories - so if you are running the `geneos
init` command as a newly created user called `geneos` should work as
expected.

If you want to use an existing directory, including adopting an existing
installation, then use the `--force`/`-F` option. The `init` command
will create any missing directories but will not remove or change
existing files. If you adopt an existing installation then see the `init
templates` command for how to add default template files to the
installation for use by new instances.

To specify a directory pass it as the only argument. It must be an
absolute path.

Note: The `init` commands no longer support setting a `USERNAME` and
will return an error. All commands must be run as the user that will own
and manage the Geneos environment.

To also add Geneos components and install software releases you can use
one of the pre-defined commands:

`package init demo`

`package init san`

`package init floating`

`package init all`

`package init template`

...

## Adopting An Existing Installation

If you have an existing Geneos installation that you manage with the
command like `gatewayctl`/`netprobectl`/etc. then you can use `geneos`
to manage those once you have set the path to the Geneos installation.

| :warning: WARNING | |:----------------------------| | `geneos` ignores
any changes to the global `.rc` files in your | | existing installation.
You **must** check and adjust individual instance | | settings to
duplicate settings. This can sometimes be very simple, for | | example
if your `netprobectl.rc` files contains a line that sets | | `JAVA_HOME`
then you can set this across all the Netprobes using `geneos | | set
netprobe -e JAVA_HOME=/path/to/java`. More complex changes, such as | |
library paths, will need careful consideration |

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
