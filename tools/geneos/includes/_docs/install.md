Install include files embedded in the `geneos` executable or from a local `.tar.gz` archive to the selected host. With no archive argument, the embedded files and configuration templates are installed. `extract` is an alias for `install` and accepts the same arguments and options.

`--prefix` defaults to `itrs` for embedded installation. Archives identify their own source when the flag is omitted, preserving existing archive installation commands. Supply `--prefix holidays` to require a holiday archive; a mismatched source fails before any destination files or links change. Only ITRS includes are embedded, so selecting another prefix without an archive is an error. The flag selects or checks the source; it does not rename files or download archives.

The archive must contain a single top-level directory named `geneos-includes-<version>` (for example `geneos-includes-v1.31.0`). Inside it, one directory named `<source>-<version>` (for example `itrs-v1.31.0` or `acme-ops-v1.31.0-rc.1`) contains `<source>.*.xml` functional include files. The prefix identifies the include **source**, independently of the deployment's site. Source names use letters, digits, dots, underscores or hyphens and begin with a letter or digit. The suffix is a `v`-prefixed semantic version, including optional prerelease and build metadata. Legacy unprefixed version directories remain supported as the `itrs` source.

Unversioned `config.*.xml` templates belong in a separate `config/` directory; `config` is reserved and cannot be a functional include source. The standalone `scripts/install-includes.sh` and `docs/` Markdown/PDF documentation may also be included. The wrapper's version must match the semantic-version suffix of the contents directory. Links, duplicate files, unexpected entries and unsafe paths are rejected before installation begins. Installation strips only the wrapper, preserving the full source-prefixed directory name.

Managed includes are installed under `gateway/includes/<source>-<version>`, including when that directory already exists. Embedded includes use the `itrs` source. Changed files are backed up to timestamped `.bak` files; unchanged files are left in place.

The optional `geneos-holidays-include-v<version>.tar.gz` archive uses the same
`geneos-includes-<version>` wrapper, with `holidays-<version>/holidays.*.xml`
contents and separate holiday documentation. Install it explicitly with
`geneos include install ARCHIVE`; holidays are not embedded in the executable
or installed by initialization. Use `includes update --prefix holidays` for
the running release version, or add `--version v<version>` for another
installed holiday version.

Both standard and holiday archives include `scripts/install-includes.sh` for
installation without `geneos`. It creates a new source-version directory and
only missing links; it refuses an existing version. It processes site
configuration only when the extracted archive has a `config/` directory, so
holiday installation does not create or migrate site templates. Run it from
the extracted archive wrapper with the Geneos home as its argument.

Configuration templates are installed under `gateway/<site>/<site>-config.*.xml`, next to `gateway/includes`. Choose a site name representing the company, customer, or client using the deployment, rather than a region or environment; for example, use `geneos config set site=acme` for Acme Corp. The `-config` infix distinguishes local overrides from functional includes when source and site have the same name. The archive retains its original `config/` and `config.*.xml` names. Templates from different sources share this site directory; use distinct template basenames to avoid sharing a destination.

> [!NOTE]
> Please note the following when choosing and using a site name:
>
> - The site name is currently used for Gateway configuration include directories and filename prefixes, including paths in supplied profiles, but future releases may use it for other features.
>
> - Spaces become underscores in paths. The resulting name must be a safe single directory path component and cannot conflict with Gateway directories such as `config` or `includes`.
>
> - If no site name is set, the directory and prefix are `local`.
>
> - Only one site name is supported per installation.
>
> - Use the same site name on all Geneos servers for consistent paths.
>
> - Changing the site name later can have unforeseen effects: existing Gateway setups and user profile files are not rewritten. Review them and run `geneos includes install` to populate the new site directory.

Before installing templates, files under `gateway/includes/config` that are in active use by a Gateway setup on the selected host, as shown by `geneos includes list --gateway`, are copied to the new site directory without replacing existing destinations. Unused files and remote includes are not copied. `config.*` filenames gain a `<site>-` prefix, and subdirectories are created as needed. Referenced files using the previous `gateway/<site>/<site>.*.xml` naming are also copied to `<site>-config.*.xml`. Originals are left intact for older Gateway setups. Existing symlinks are copied as regular files containing their targets' contents; broken links cause an error. Templates are installed only when no destination exists. Run `geneos includes fix` to migrate references where old and new contents match.

The standalone `scripts/install-includes.sh` cannot make this check without the `geneos` command and so still copies every file from the old directory.

Missing top-level include links are created automatically. Existing files and links are left alone unless `--update` (`-U`) is given, which replaces them with links to the installed version. When an archive is supplied, updates target the version in the archive; otherwise they target the version embedded in the executable.

With `--update`, a preflight checks obsolete links against gateway setups before any links are changed, as for `includes update`. Cleanup is restricted to the installed source; other sources' links are retained. A replacement link belonging to another source is an error. Same-basename replacement files are otherwise valid updates, regardless of the old target directory. If any obsolete link without a replacement is referenced, the update fails with all links unchanged; extracted files remain installed. After a successful preflight, links are updated and unused obsolete links for that source are removed. Without `--update`, no obsolete links are removed.

```shell
geneos includes install
geneos includes install --update
geneos includes install geneos-includes-v1.31.0.tar.gz
geneos includes install --update geneos-includes-v1.31.0.tar.gz
geneos include install geneos-holidays-include-v1.31.0.tar.gz
geneos include install --prefix holidays --update geneos-holidays-include-v1.31.0.tar.gz
```

The archive is read locally; the global `--host` (`-H`) option selects the installation destination. Completed filesystem operations are not rolled back if a later operation fails.

Installation reports extracted or unchanged versioned files, extracted or preserved site templates, copied or skipped legacy configuration files, and created, updated or preserved top-level include paths. Existing site templates are reported as skipped, not compared or overwritten.
