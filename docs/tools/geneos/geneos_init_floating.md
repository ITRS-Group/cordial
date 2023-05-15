## geneos init floating

Initialise a Geneos Floating Netprobe environment

### Synopsis


Install a Floating Netprobe into a new Geneos install
directory.

Without any flags the command installs a Floating Netprobe in a directory called
`geneos` under the user's home directory (unless the user's home
directory ends in `geneos` in which case it uses that directly),
downloads the latest netprobe release and create a netprobe instance using
the `hostname` of the system.

In almost all cases authentication will be required to download the
Netprobe package and as this is a new Geneos installation it is
unlikely that the download credentials are saved in a local config
file, so use the `-u email@example.com` as appropriate.

If you have a netprobe software archive locally then use the `-A
PATH`. If the name of the file is not in the same format as
downloaded from the official site(s) then you have to also set the
type (netprobe) and version using the `-T [TYPE:]VERSION`. TYPE is
set to `netprobe` if not given. 

The initial configuration file is built from the default templates
installed and located in `.../templates` but this can be overridden
with the `-s` option. You can set `gateways`, `types`, `attributes`,
`variables` using the appropriate flags. These flags can be specified
multiple times.


```
geneos init floating [flags] [USERNAME] [DIRECTORY]
```

### Options

```
  -V, --version VERSION              Download this VERSION, defaults to latest. Doesn't work for EL8 archives. (default "latest")
  -A, --archive PATH or URL          PATH or URL to software archive to install
  -T, --override [TYPE:]VERSION      Override the [TYPE:]VERSION for archive files with non-standard names
  -g, --gateway HOSTNAME:PORT        Add gateway in the format NAME:PORT. Repeat flag for more gateways.
  -a, --attribute NAME=VALUE         Add an attribute in the format NAME=VALUE. Repeat flag for more attributes.
  -t, --type NAME                    Add a type NAME. Repeat flag for more types.
  -v, --variable [TYPE:]NAME=VALUE   Add a variable in the format [TYPE:]NAME=VALUE. Repeat flag for more variables
```

### Options inherited from parent commands

```
  -G, --config string                config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -e, --env NAME=VALUE               Add an environment variable in the format NAME=VALUE. Repeat flag for more values.
  -f, --floatingtemplate string      Floating probe template file
  -F, --force                        Be forceful, ignore existing directories.
  -w, --gatewaytemplate string       A gateway template file
  -c, --importcert string            signing certificate file with optional embedded private key
  -k, --importkey string             signing private key file
  -l, --log                          Run 'logs -f' after starting instance(s)
  -C, --makecerts                    Create default certificates for TLS support
  -n, --name string                  Use the given name for instances and configurations instead of the hostname
  -N, --nexus                        Download from nexus.itrsgroup.com. Requires ITRS internal credentials
  -s, --santemplate string           SAN template file
  -p, --snapshots                    Download from nexus snapshots. Requires -N
  -u, --username download.username   Username for downloads. Defaults to configuration value download.username
```

### SEE ALSO

* [geneos init](geneos_init.md)	 - Initialise a Geneos installation

