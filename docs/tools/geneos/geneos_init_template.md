## geneos init template

Initialise or overwrite templates

### Synopsis


The `geneos` commands contains a number of default template files
that are normally written out during initialization of a new
installation. In the case of adopting a legacy installation or
upgrading the program it might be desirable to extract these template
files.

This command will overwrite any files with the same name but will not
delete other template files that may already exist.

Use this command if you get missing template errors using the `add`
command.


```
geneos init template [flags]
```

### Options inherited from parent commands

```
  -G, --config string                config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -e, --env NAME=VALUE               Add an environment variable in the format NAME=VALUE. Repeat flag for more values.
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

