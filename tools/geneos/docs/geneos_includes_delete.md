# `geneos includes delete`

Delete unused version directories and their top-level symlinks from `gateway/includes` on the selected host. The `remove` alias has the same behaviour.

`--prefix` selects the include **source** and defaults to `itrs`, including legacy unprefixed semantic-version directories. Use `--prefix holidays` for holiday versions, or quote `--prefix '*'` to select all sources. This source filter also applies to explicit version patterns, so a broad pattern cannot remove another source accidentally.

With no version pattern, all unused versions of the selected source are removed and versions referenced by gateways are skipped. To select versions, supply one or more shell-style patterns as positional arguments, or one pattern using `--version` (`-V`). Patterns match full directory names, including the source and `v` prefix, and support `*`, `?`, and character classes such as `[12]`. Quote patterns so the shell does not expand them. Supplying positional arguments together with the flag is an error.

Before removing any version, the command checks every symlink directly in the parent `gateway/includes` directory and loads gateway setups using the same loader as `list --gateway`. All links for a version are checked before any are removed. A gateway reference to even one link or directly into the version protects every link and the entire version directory. Include groups, disabled and optional includes, relative and absolute targets, link chains, directory aliases, and dangling references into the version are checked.

Positional patterns are processed in order. If a pattern selects any referenced version, none of the versions or links matching that pattern are deleted, and an error identifies the version and its protecting reference. Later patterns are still processed, and collected errors are returned after the list is complete. If a gateway setup cannot be loaded or a link cannot be inspected or resolved safely, that pattern fails without proceeding with deletion.

Links originating elsewhere are not removed: those are manually managed by the user, although gateway references through them still protect the version. Configuration templates, non-version directories, and version-directory symlinks are never deleted. Missing includes directories and patterns that match no versions are harmless no-ops.

```shell
geneos includes delete
geneos includes delete 'itrs-v1.30.*'
geneos includes delete --prefix holidays 'holidays-v2026.01.*'
geneos includes delete --prefix '*' 'itrs-v1.29.*' 'acme-v1.30.*'
geneos includes delete --version 'itrs-v1.31.0-rc.*'
geneos includes remove -V 'v1.29.*' # legacy directories
```

Use the global `--host` (`-H`) option to select the installation destination. Links are checked again immediately before each directory is removed. Deleted directories are not backed up; earlier deletions are not rolled back if a later filesystem operation fails or a new protecting link appears during deletion.

## Usage

```text
geneos includes delete [VERSION_PATTERN...] [flags]
```

### Options

```text
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/geneos/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --prefix string             Select the include source prefix, or '*' for all sources (default "itrs")
      --profiles string           profiles configuration (defaults are ~/.config/geneos/profiles.yaml, /etc/geneos/profiles.yaml)
  -V, --version string            Select include versions using a shell-style pattern
```

## SEE ALSO

* [geneos includes](geneos_includes.md)	 - Manage includes
