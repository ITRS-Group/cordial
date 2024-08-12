The `package uninstall` commands removes installed Geneos releases and downloaded archive files. By default all releases that are not used by any enabled or running instance are removed with the exception of the "latest" release for each matching component.

To keep the downloaded archives (in `${GENEOS}/packages/download`) use the `--keep`/`-k` flag, otherwise **all** files in that directory are removed, including the latest release and those that are for other components.

If `TYPE` is given then only releases for that component are removed. Similarly, if `--version VERSION` is given then only that version is removed. `VERSION` must be an exact match and multiple versions or version wildcards are not yet supported.

Note that if `TYPE` is for a component type that uses a different underlying release, such as a `san` which could be a `netprobe` or `fa2` under the hood, you have to remove the main `TYPE`.

To remove releases that are in use by protected instances you must give the `--force`/`-F` flag.

To update releases in use by instances, whether running or not, use the `--update`/`-U` flag. Base links are only updated if this flag is given (and `--force` if any instance using it is marked protected) unless all matching instances are disabled. For each release being removed any running instances will first be stopped and base links will be updated to point to the "latest" version (unless the `--all` flag is used). Any instances stopped will be restarted after all other actions are complete.

If the `-all` flag is passed then all matching releases are removed and all running instances stopped and disabled. This can be used to force a "clean install" of a component or before removal of a Geneos installation on a specific host.

If a host is not selected with the `--host HOST` flags then the uninstall applies to all configured hosts. 

Use `geneos package list` to see which releases are installed.
