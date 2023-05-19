# `geneos init demo`

Initialise a Geneos Demo environment

```text
geneos init demo [flags] [USERNAME] [DIRECTORY]
```

## Details

Initialise a Geneos Demo environment, creating a new directory structure
as required.

Without any flags the command installs the components in a directory
called `geneos` under the user's home directory (unless the user's home
directory ends in `geneos` in which case it uses that directly),
downloads the latest release archives and creates a Gateway instance
using the name `Demo Gateway` (with embedded space) as required for Demo
licensing, as Netprobe and a Webserver.

If the release archive files required have already been downloaded then
use the `-A directory` flag to indicate their location. For each
component type this directory is checked for the latest release.

Otherwise, to fetch the releases from the ITRS download server
authentication will be required use the `-u email@example.com` to
specify the user account and you will be prompted for a password.

The initial configuration file for the Gateway is built from the default
templates installed and located in `.../templates` but this can be
overridden with the `-s` option. For the Gateway you can add include
files using `-i PRIORITY:PATH` flag. This can be repeated multiple
times.

Other flags inherited from the `geneos init` command can be used to
influence the installation.

### Options

```text
  -A, --archive string                Directory of releases for installation
  -i, --include PRIORITY:{URL|PATH}   A gateway connection in the format HOSTNAME:PORT
                                      (Repeat as required, san and floating only)
```

### Options inherited from parent commands

```text
  -G, --config string             config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -e, --env NAME=VALUE            An environment variable for instance start-up
                                  (Repeat as required)
  -f, --floatingtemplate string   Floating probe template file
  -F, --force                     Be forceful, ignore existing directories.
  -w, --gatewaytemplate string    A gateway template file
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters)
  -c, --importcert string         signing certificate file with optional embedded private key
  -k, --importkey string          signing private key file
  -l, --log                       Follow logs after starting instance(s)
  -C, --makecerts                 Create default certificates for TLS support
  -n, --name string               Use name for instances and configurations instead of the hostname
  -N, --nexus                     Download from nexus.itrsgroup.com. Requires ITRS internal credentials
  -s, --santemplate string        SAN template file
  -p, --snapshots                 Download from nexus snapshots. Requires -N
  -u, --username string           Username for downloads
  -V, --version string            Download matching version, defaults to latest. Doesn't work for EL8 archives. (default "latest")
```

## SEE ALSO

* [geneos init](geneos_init.md)	 - Initialise a Geneos installation
