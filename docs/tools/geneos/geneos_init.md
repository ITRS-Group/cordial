## geneos init

Initialise a Geneos installation

### Synopsis


Initialise a Geneos installation by creating the directory
hierarchy and user configuration file, with the USERNAME and
DIRECTORY if supplied. DIRECTORY must be an absolute path and
this is used to distinguish it from USERNAME.

**Note**: This command has too many options and flags and will be
replaced by a number of sub-commands that will narrow down the flags
and options required. Backward compatibility will be maintained as
much as possible but top-level `init` flags may be hidden from usage
messages.

DIRECTORY defaults to `${HOME}/geneos` for the selected user unless
the last component of `${HOME}` is `geneos` in which case the home
directory is used. e.g. if the user is `geneos` and the home
directory is `/opt/geneos` then that is used, but if it were a
user `itrs` which a home directory of `/home/itrs` then the
directory `/home/itrs/geneos` would be used. This only applies
when no DIRECTORY is explicitly supplied.

When DIRECTORY is given it must be an absolute path and the
parent directory must be writable by the user - either running
the command or given as USERNAME.

DIRECTORY, whether explicit or implied, must not exist or be
empty of all except "dot" files and directories.

When run with superuser privileges a USERNAME must be supplied
and only the configuration file for that user is created. e.g.:

	sudo geneos init geneos /opt/itrs

When USERNAME is supplied then the command must either be run
with superuser privileges or be run by the same user.

Any PARAMS provided are passed to the 'add' command called for
components created.


```
geneos init [flags] [USERNAME] [DIRECTORY] [PARAMS]
```

### Examples

```

geneos init # basic set-up and user config file
geneos init -D -u email@example.com # create a demo environment, requires password
geneos init -S -n mysan -g Gateway1 -t App1Mon -a REGION=EMEA # install and run a SAN

```

### Options

```
  -c, --importcert string             signing certificate file with optional embedded private key
  -k, --importkey string              signing private key file
  -w, --gatewaytemplate string        A gateway template file
  -s, --santemplate string            A san template file
  -e, --env NAME                      (all components) Add an environment variable in the format NAME=VALUE
  -i, --include PRIORITY:{URL|PATH}   (gateways) Add an include file in the format PRIORITY:PATH
  -g, --gateway HOSTNAME:PORT         (sans) Add a gateway in the format NAME:PORT
  -a, --attribute NAME                (sans) Add an attribute in the format NAME=VALUE
  -t, --type NAME                     (sans) Add a gateway in the format NAME:PORT
  -v, --variable [TYPE:]NAME=VALUE    (sans) Add a variable in the format [TYPE:]NAME=VALUE
  -C, --makecerts                     Create default certificates for TLS support
  -l, --log                           Run 'logs -f' after starting instance(s)
  -F, --force                         Be forceful, ignore existing directories.
  -n, --name string                   Use the given name for instances and configurations instead of the hostname
  -N, --nexus                         Download from nexus.itrsgroup.com. Requires auth.
  -p, --snapshots                     Download from nexus snapshots (pre-releases), not releases. Requires -N
  -V, --version string                Download matching version, defaults to latest. Doesn't work for EL8 archives. (default "latest")
  -u, --username string               Username for downloads. Defaults to configuration value download.username
  -P, --pwfile string                 
  -h, --help                          help for init
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
* [geneos init all](geneos_init_all.md)	 - Initialise a complete Geneos environment
* [geneos init demo](geneos_init_demo.md)	 - Initialise a Geneos Demo environment
* [geneos init san](geneos_init_san.md)	 - Initialise a Geneos SAN (Self-Announcing Netprobe) environment
* [geneos init template](geneos_init_template.md)	 - Initialise or overwrite templates

