# `geneos includes install`

Install include files embedded in the `geneos` executable or from a local `.tar.gz` archive to the selected host. With no archive argument, the embedded files and configuration templates are installed. `extract` is an alias for `install` and accepts the same arguments and options.

The archive must contain one directory named with a `v`-prefixed semantic version (for example `v1.31.0` or `v1.31.0-rc.1`), containing `itrs.*.xml` include files. Unversioned `config.*.xml` templates belong in a separate `config/` directory. Links, duplicate files, unexpected entries and unsafe paths are rejected before installation begins.

Managed includes are installed under `gateway/includes/<version>`, including when that directory already exists. Changed files are backed up to timestamped `.bak` files; unchanged files are left in place.

Configuration templates are installed under `gateway/includes/config` only when no path of the same name exists. Local configuration files and existing symlinks are never replaced.

Missing top-level include links are created automatically. Existing files and links are left alone unless `--update` (`-U`) is given, which replaces them with links to the installed version. When an archive is supplied, updates target the version in the archive; otherwise they target the version embedded in the executable.

```shell
geneos includes install
geneos includes install --update
geneos includes install cordial-includes-v1.31.0.tar.gz
geneos includes install --update cordial-includes-v1.31.1.tar.gz
```

The archive is read locally; the global `--host` (`-H`) option selects the installation destination. Completed filesystem operations are not rolled back if a later operation fails.

## Usage

```text
geneos includes install [ARCHIVE.tar.gz] [flags]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
  -U, --update            Update version links for existing include files
```

## SEE ALSO

* [geneos includes](geneos_includes.md)	 - Manage includes
