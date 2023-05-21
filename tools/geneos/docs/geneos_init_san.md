# `geneos init san`

Initialise a Geneos SAN (Self-Announcing Netprobe) environment

```text
geneos init san [flags] [USERNAME] [DIRECTORY]
```

Install a Self-Announcing Netprobe (SAN) into a new Geneos install
directory.

Without any flags the command installs a SAN in a directory called
`geneos` under the user's home directory (unless the user's home
directory ends in `geneos` in which case it uses that directly),
downloads the latest netprobe release and create a SAN instance using
the `hostname` of the system.

In almost all cases authentication will be required to download the
Netprobe package and as this is a new Geneos installation it is unlikely
that the download credentials are saved in a local config file, so use
the `-u email@example.com` as appropriate.

If you have a netprobe software archive locally then use the `-A PATH`.
If the name of the file is not in the same format as downloaded from the
official site(s) then you have to also set the type (netprobe) and
version using the `-T [TYPE:]VERSION`. TYPE is set to `netprobe` if not
given. 

The initial configuration file is built from the default templates
installed and located in `.../templates` but this can be overridden with
the `-s` option. You can set `gateways`, `types`, `attributes`,
`variables` using the appropriate flags. These flags can be specified
multiple times.

### Options

```text
  -V, --version VERSION              Download this VERSION, defaults to latest. Doesn't work for EL8 archives. (default "latest")
  -A, --archive string               Directory of releases for installation
  -T, --override [TYPE:]VERSION      Override the [TYPE:]VERSION for archive files with non-standard names
  -g, --gateway HOSTNAME:PORT        A gateway connection in the format HOSTNAME:PORT
                                     (Repeat as required, san and floating only)
  -a, --attribute NAME=VALUE         An attribute in the format NAME=VALUE
                                     (Repeat as required, san only)
  -t, --type NAME                    A type NAME
                                     (Repeat as required, san only)
  -v, --variable [TYPE:]NAME=VALUE   A variable in the format [TYPE:]NAME=VALUE
                                     (Repeat as required, san only)
```

### Options inherited from parent commands

```text
  -G, --config string             config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -e, --env NAME=VALUE            An environment variable for instance start-up
                                  (Repeat as required)
  -f, --floatingtemplate string   Floating probe template file
  -F, --force                     Be forceful, ignore existing directories.
  -w, --gatewaytemplate string    A gateway template file
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
  -c, --importcert string         signing certificate file with optional embedded private key
  -k, --importkey string          signing private key file
  -l, --log                       Follow logs after starting instance(s)
  -C, --makecerts                 Create default certificates for TLS support
  -n, --name string               Use name for instances and configurations instead of the hostname
  -N, --nexus                     Download from nexus.itrsgroup.com. Requires ITRS internal credentials
  -s, --santemplate string        SAN template file
  -p, --snapshots                 Download from nexus snapshots. Requires -N
  -u, --username string           Username for downloads
```

## SEE ALSO

* [geneos init](geneos_init.md)	 - Initialise a Geneos installation
