Install include files from a local `.tar.gz` archive to the selected host.

The archive must contain one directory named with a `v`-prefixed semantic version (for example `v1.31.0` or `v1.31.0-rc.1`), containing `itrs.*.xml` include files. Unversioned `config.*.xml` templates belong in a separate `config/` directory. Links, duplicate files, unexpected entries and unsafe paths are rejected before installation begins.

Managed includes are installed under `gateway/includes/<version>`, including when that directory already exists. Changed files are backed up to timestamped `.bak` files; unchanged files are left in place.

Configuration templates are installed under `gateway/includes/config` only when no path of the same name exists. Local configuration files and existing symlinks are never replaced.

Missing top-level include links are created automatically. Existing files and links are left alone unless `--update` (`-U`) is given, which replaces them with links to the installed version. Updates target the version in the archive, not the version embedded in the executable.

```shell
geneos includes install cordial-includes-v1.31.0.tar.gz
geneos includes install --update cordial-includes-v1.31.1.tar.gz
```

The archive is read locally; the global `--host` (`-H`) option selects the installation destination. Completed filesystem operations are not rolled back if a later operation fails.
