# `geneos init all`

Initialise a more complete Geneos environment

```text
geneos init all [flags] [USERNAME] [DIRECTORY]
```

Initialise a typical Geneos installation.

This command initialises a Geneos installation by:
- Creating the directory structure & user configuration file,
- Installing software packages for component types `gateway`, `licd`,
  `netprobe` & `webserver`,
- Creating an instance for each component type named after the hostname
  (except for `netprobe` whose instance is named `localhost`)
- Starting the created instances.

A license file is required and should be given using option `-L`. If a
license file is not available, then use `-L /dev/null` which will create
an empty `geneos.lc` file that can be overwritten later.

Authentication will most-likely be required to download the installation
software packages and, as this is a new Geneos installation, it is
unlikely that the download credentials are saved in a local config file.
Use option `-u email@example.com` to define the username for downloading
software packages.

If packages are already downloaded locally, use option `-A
Path_To_Archive` to refer to the directory containing the package
archives.  Package files must be named in the same format as those
downloaded from the [ITRS download
portal](https://resources.itrsgroup.com/downloads). If no version is
given using option `-V`, then the latest version of each component is
installed.

### Options

```text
  -L, --licence string                Licence file location (default "geneos.lic")
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

## Examples

```bash
geneos init all -L https://myserver/files/geneos.lic -u email@example.com
geneos init all -L ~/geneos.lic -A ~/downloads /opt/itrs
sudo geneos init all -L /tmp/geneos-1.lic -u email@example.com myuser /opt/geneos

```

## SEE ALSO

* [geneos init](geneos_init.md)	 - Initialise a Geneos installation
