# `geneos package uninstall`

Uninstall Geneos releases

```text
geneos package uninstall [flags] [TYPE]
```

The `package uninstall` commands removes installed Geneos releases. By default all releases that are not used by any enabled or running instance are removed with the exception of the "latest" release.

If `TYPE` is given then only releases for that component are removed. Similarly, if `--version VERSION` is given then only that version is removed. `VERSION` must be an exact match and multiple versions or version wildcards are not yet supported.

Note that if `TYPE` is for a component type that uses a different underlying release, such as a `san` which could be a `netprobe` or `fa2` under the hood, you have to remove the main `TYPE`.

To remove releases that are in use by protected instances you must give the `--force` flag.

For each release being removed any running instances will first be stopped and base links will be updated to point to the "latest" version (unless the `--all` flag is used). Any instances stopped will be restarted after all other actions are complete.

If the `-all` flag is passed then all matching releases are removed and all running instances stopped and disabled. This can be used to force a "clean install" of a component or before removal of a Geneos installation on a specific host.

If a host is not selected with the `--host HOST` flags then the uninstall applies to all configured hosts. 

Use `geneos update list` to see which releases are installed.

### Options

```text
  -V, --version VERSION   Uninstall VERSION
  -A, --all               Uninstall all releases, stopping and disabling running instances
  -f, --force             Force uninstall, stopping protected instances first
```

## Examples

```bash
geneos uninstall netprobe
geneos uninstall --version 5.14.1

```

## SEE ALSO

* [geneos package](geneos_package.md)	 - Package Operations
