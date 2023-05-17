## geneos init all

Initialise a more complete Geneos environment

### Synopsis


Initialise a typical Geneos installation.

This command initialises a Geneos installation by:
- Creating the directory structure & user configuration file,
- Installing software ackages for component types `gateway`, `licd`,
  `netprobe` & `webserver`,
- Creating an instance for each component type named after the hostname
  (except for `netprobe` whose instance is named `localhost`)
- Starting the created instances.

A license file is required and should be given using option `-L`.
If a license file is not available, then use `-L /dev/null` which will
create an empty `geneos.lc` file that can be overwritten later.

Authentication will most-likely be required to download the installation
software packages and, as this is a new Geneos installation, it is unlikely
that the download credentials are saved in a local config file.
Use option `-u email@example.com` to define the username for downloading
software packages.

If packages are already downloaded locally, use option `-A Path_To_Archive`
to refer to the directory containing the package archives.  Package files
must be named in the same format as those downloaded from the 
[ITRS download portal](https://resources.itrsgroup.com/downloads).
If no version is given using option `-V`, then the latest version of each
component is installed.


```
geneos init all [flags] [USERNAME] [DIRECTORY]
```

### Examples

```

geneos init all -L https://myserver/files/geneos.lic -u email@example.com
geneos init all -L ~/geneos.lic -A ~/downloads /opt/itrs
sudo geneos init all -L /tmp/geneos-1.lic -u email@example.com myuser /opt/geneos

```

### Options

```
  -L, --licence Filepath or URL       Filepath or URL to license file (default "geneos.lic")
  -A, --archive PATH or URL           PATH or URL to software archive to install
  -i, --include PRIORITY:{URL|PATH}   (gateways) Add an include file in the format PRIORITY:PATH
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
  -V, --version string               Download matching version, defaults to latest. Doesn't work for EL8 archives. (default "latest")
```

### SEE ALSO

* [geneos init](geneos_init.md)	 - Initialise a Geneos installation
