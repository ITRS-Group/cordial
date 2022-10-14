## geneos init template

Initialise or overwrite templates

### Synopsis




```
geneos init template [flags]
```

### Options

```
  -h, --help   help for template
```

### Options inherited from parent commands

```
  -G, --config string     config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -F, --force             Be forceful, ignore existing directories.
  -l, --log               Run 'logs -f' after starting instance(s)
  -C, --makecerts         Create default certificates for TLS support
  -n, --name string       Use the given name for instances and configurations instead of the hostname
  -N, --nexus             Download from nexus.itrsgroup.com. Requires auth.
  -P, --pwfile string     
  -q, --quiet             quiet mode
  -p, --snapshots         Download from nexus snapshots (pre-releases), not releases. Requires -N
  -u, --username string   Username for downloads. Defaults to configuration value download.username
  -V, --version string    Download matching version, defaults to latest. Doesn't work for EL8 archives. (default "latest")
```

### SEE ALSO

* [geneos init](geneos_init.md)	 - Initialise a Geneos installation

