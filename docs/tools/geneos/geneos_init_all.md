## geneos init all

Initialise a more complete Geneos environment

### Synopsis


Initialise a typical Geneos installation.

This command installs a Gateway, Netprobe, Licence-Daemon and
Webserver. A licence file is required and should be give using the
`-L` flag. If a licence file is not available then use `-L /dev/null`
which will create and empty `geneos.lic` file that can be overwritten
later.

In almost all cases authentication will be required to download the
install packages and as this is a new Geneos installation it is
unlikely that the download credentials are saved in a local config
file, so use the `-u email@example.com` as appropriate.

If packages are already downloaded locally then use the `-A` flag to
refer to the directory contain the archives. They must be named in
the same format as those downloaded from the main download website.
If no version is given using the `-V` flag then the latest version of
each component is installed.



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

