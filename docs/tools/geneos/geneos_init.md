## geneos init

Initialise a Geneos installation

### Synopsis


Initialise a Geneos installation by creating the directory hierarchy
and user configuration file, with the USERNAME and DIRECTORY if
supplied. DIRECTORY must be an absolute path and this is used to
distinguish it from USERNAME.

**Note**: This command has too many flags and options and has been
supplemented by a number of sub-commands that do more specific
things. Backward compatibility will be maintained as much as possible
but top-level `init` flags may be hidden from usage messages.

Please see the sub-commands below for a more appropriate command.

DIRECTORY defaults to `${HOME}/geneos` for the selected user unless
the last component of `${HOME}` is `geneos` in which case the home
directory is used. e.g. if the user is `geneos` and the home
directory is `/opt/geneos` then that is used, but if it were a user
`itrs` which a home directory of `/home/itrs` then the directory
`/home/itrs/geneos` would be used. This only applies when no
DIRECTORY is explicitly supplied.

When DIRECTORY is given it must be an absolute path and the parent
directory must be writable by the user - either running the command
or given as USERNAME.

DIRECTORY, whether explicit or implied, must either not exist or be
empty except for "dot" files.

When run with superuser privileges a USERNAME must be supplied and
only the configuration file for that user is created. e.g.:

	sudo geneos init geneos /opt/itrs

When USERNAME is supplied then the command must either be run
with superuser privileges or be run by the same user.


```
geneos init [flags] [USERNAME] [DIRECTORY]
```

### Examples

```

# creates an Geneos tree under home area
geneos init
# to create new directory as `geneos`
sudo geneos init geneos /opt/itrs

```

### Options

```
  -C, --makecerts                    Create default certificates for TLS support
  -l, --log                          Run 'logs -f' after starting instance(s)
  -F, --force                        Be forceful, ignore existing directories.
  -n, --name string                  Use the given name for instances and configurations instead of the hostname
  -c, --importcert string            signing certificate file with optional embedded private key
  -k, --importkey string             signing private key file
  -N, --nexus                        Download from nexus.itrsgroup.com. Requires ITRS internal credentials
  -p, --snapshots                    Download from nexus snapshots. Requires -N
  -V, --version string               Download matching version, defaults to latest. Doesn't work for EL8 archives. (default "latest")
  -u, --username download.username   Username for downloads. Defaults to configuration value download.username
  -w, --gatewaytemplate string       A gateway template file
  -s, --santemplate string           A san template file
  -e, --env NAME=VALUE               Add an environment variable in the format NAME=VALUE. Repeat flag for more values.
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
* [geneos init all](geneos_init_all.md)	 - Initialise a more complete Geneos environment
* [geneos init demo](geneos_init_demo.md)	 - Initialise a Geneos Demo environment
* [geneos init san](geneos_init_san.md)	 - Initialise a Geneos SAN (Self-Announcing Netprobe) environment
* [geneos init template](geneos_init_template.md)	 - Initialise or overwrite templates

