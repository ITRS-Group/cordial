## geneos init

Initialise a Geneos installation

### Synopsis


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


```
geneos init [flags] [USERNAME] [DIRECTORY]
```

### Examples

```

# To create a Geneos tree under home area
geneos init
# To create a new Geneos tree owned by user `geneos` under `/opt/itrs`
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

