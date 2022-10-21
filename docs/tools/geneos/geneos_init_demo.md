## geneos init demo

Initialise a Geneos Demo environment

### Synopsis


Install a Demo environment into a new Geneos install directory
layout.

Without any flags the command installs the components in a directory
called `geneos` under the user's home directory (unless the user's
home directory ends in `geneos` in which case it uses that directly),
downloads the latest release archives and creates a Gateway instance
using the name `Demo` as required for Demo licensing, as Netprobe and
a Webserver.

In almost all cases authentication will be required to download the
install packages and as this is a new Geneos installation it is
unlikely that the download credentials are saved in a local config
file, so use the `-u email@example.com` as appropriate.

The initial configuration file for the Gateway is built from the
default templates installed and located in `.../templates` but this
can be overridden with the `-s` option. For the Gateway you can add
include files using `-i PRIORITY:PATH` flag. This can be repeated
multiple times.

The `-e` flag adds environment variables to all instances created and
so should only be used for common values, such as `TZ`.


```
geneos init demo [flags] [USERNAME] [DIRECTORY]
```

### Options

```
  -A, --archive PATH or URL           PATH or URL to software archive to install
  -e, --env NAME=VALUE                Add environment variables in the format NAME=VALUE. Repeat flag for more values.
  -i, --include PRIORITY:{URL|PATH}   (gateways) Add an include file in the format PRIORITY:PATH
```

### Options inherited from parent commands

```
  -G, --config string                config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -F, --force                        Be forceful, ignore existing directories.
  -w, --gatewaytemplate string       A gateway template file
  -c, --importcert string            signing certificate file with optional embedded private key
  -k, --importkey string             signing private key file
  -l, --log                          Run 'logs -f' after starting instance(s)
  -C, --makecerts                    Create default certificates for TLS support
  -n, --name string                  Use the given name for instances and configurations instead of the hostname
  -N, --nexus                        Download from nexus.itrsgroup.com. Requires ITRS internal credentials
  -s, --santemplate string           A san template file
  -p, --snapshots                    Download from nexus snapshots. Requires -N
  -u, --username download.username   Username for downloads. Defaults to configuration value download.username
  -V, --version string               Download matching version, defaults to latest. Doesn't work for EL8 archives. (default "latest")
```

### SEE ALSO

* [geneos init](geneos_init.md)	 - Initialise a Geneos installation

