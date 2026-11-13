Install include files embedded in the `geneos` executable or from a local `.tar.gz` archive to the selected host. With no archive argument, the embedded files and configuration templates are installed. `extract` is an alias for `install` and accepts the same arguments and options.

The archive must contain a single top-level directory named `geneos-includes-<version>` (for example `geneos-includes-v1.31.0`). Inside it, one directory named with a `v`-prefixed semantic version (for example `v1.31.0` or `v1.31.0-rc.1`) contains `itrs.*.xml` include files. Unversioned `config.*.xml` templates belong in a separate `config/` directory; the standalone `scripts/install-includes.sh` may also be included. The wrapper's version must match the version directory. Links, duplicate files, unexpected entries and unsafe paths are rejected before installation begins, and `geneos includes install` strips the wrapper before staging the files.

Managed includes are installed under `gateway/includes/<version>`, including when that directory already exists. Changed files are backed up to timestamped `.bak` files; unchanged files are left in place.

Configuration templates are installed under `gateway/<site>/<site>.*.xml`, next to `gateway/includes`. Choose a site name representing the company, customer, or client using the deployment, rather than a region or environment; for example, use `geneos config set site=acme` for Acme Corp. The archive retains its original `config/` and `config.*.xml` names.

> [!NOTE]
> Please note the following when choosing and using a site name:
>
> - The site name is currently used for Gateway configuration include directories and filename prefixes, including paths in supplied profiles, but future releases may use it for other features.
> - Spaces become underscores in paths. The resulting name must be a safe single directory path component and cannot conflict with Gateway directories such as `config` or `includes`.
> - If no site name is set, the directory and prefix are `local`.
> - Only one site name is supported per installation.
> - Use the same site name on all Geneos servers for consistent paths.
> - Changing the site name later can have unforeseen effects: existing Gateway setups and user profile files are not rewritten. Review them and run `geneos includes install` to populate the new site directory.

Before installing templates, existing files under `gateway/includes/config` are copied to the new site directory without replacing existing destinations. `config.*` file prefixes become the site name; other files and subdirectories are copied as well. The original directory is left intact for older Gateway setups. Existing symlinks in the old directory are copied as regular files containing their targets' contents; broken links cause an error. Templates are installed only when no destination exists.

Missing top-level include links are created automatically. Existing files and links are left alone unless `--update` (`-U`) is given, which replaces them with links to the installed version. When an archive is supplied, updates target the version in the archive; otherwise they target the version embedded in the executable.

With `--update`, a preflight checks obsolete links against gateway setups before any links are changed, as for `includes update`. Same-basename replacement files in the installed version are valid updates, regardless of the old target directory. If any obsolete link without a replacement is referenced, the update fails with all links unchanged; extracted files remain installed. After a successful preflight, links are updated and unused obsolete links are removed. Without `--update`, no obsolete links are removed.

```shell
geneos includes install
geneos includes install --update
geneos includes install geneos-includes-v1.31.0.tar.gz
geneos includes install --update geneos-includes-v1.31.0.tar.gz
```

The archive is read locally; the global `--host` (`-H`) option selects the installation destination. Completed filesystem operations are not rolled back if a later operation fails.
