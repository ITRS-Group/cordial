## geneos package uninstall

Uninstall Geneos releases

### Synopsis


Uninstall selected Geneos releases. By default all releases that are
not used by any enabled or running instance are removed with the
exception of the "latest" release.

If `TYPE` is given then only releases for that component are
considered. Similarly, if `--version VERSION` is given then only that
version is removed. `VERSION` must be an exact match and multiple
versions or version wildcards are not yet supported.

To remove releases that are in use by protected instances you must
give the `--force` flag.

For each release being removes any running instances will first be
stopped and base links will be updated to point to the "latest"
version (unless the `--all` flag is used). Any instances stopped will
be restarted after all other actions are complete.

If the `-all` flag is passed then all matching releases are removed
and all running instances stopped and disabled. This can be used to
force a "clean install" of a component or before removal of a Geneos
installation on a specific host.

If a host is not selected with the `--host HOST` flags then the
uninstall applies to all configured hosts. 

Use `geneos update ls` to see what is installed.


```
geneos package uninstall [flags] [TYPE]
```

### Examples

```

geneos uninstall netprobe
geneos uninstall --version 5.14.1

```

### Options

```
  -A, --all              Uninstall all releases, stopping and disabling running instances
  -f, --force            Force uninstall, stopping protected instances first
  -H, --host string      Perform on a remote host. "all" means all hosts and locally (default "all")
  -V, --version string   Uninstall a specific version
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos package](geneos_package.md)	 - A brief description of your command
