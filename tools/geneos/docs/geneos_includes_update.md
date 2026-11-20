# `geneos includes update`

Update the top-level include symlinks to point to files in a versioned include directory on the selected host.

By default, the links are updated to `itrs-<version>` embedded in the running `geneos` executable. Use `--prefix` to select another include **source**, such as `holidays`. Without `--version`, the selected directory is `<prefix>-<running-geneos-version>`, not the newest installed version. Other sources must already be installed; only the current embedded ITRS version is extracted automatically when needed.

Use `--prefix holidays --version v1.31.0` to select `holidays-v1.31.0`. A version may omit its leading `v`. Full source-version directory names, such as `holidays-v1.31.0` or `acme-ops-v2.0.0-rc.1`, also remain supported without `--prefix`; an explicitly supplied conflicting prefix is an error. The final version suffix is a `v`-prefixed semantic version, including optional prerelease and build metadata. Without an explicit prefix, bare semantic versions select legacy unprefixed ITRS directories as before; with `--prefix itrs`, they select `itrs-v<version>`. A missing selected directory is an error.

```shell
geneos include update
geneos include update --prefix holidays
geneos include update --prefix holidays --version v1.31.0
geneos include update --version holidays-v1.31.0
```

Links that already point to the selected version are left unchanged. If a target path already exists with a different link or file, it is removed before creating the new symlink, except that a link owned by another source blocks the update before any links change.

Before changing any links, a preflight checks obsolete top-level symlinks using the same gateway setup loader as `list --gateway`, including nested include groups. Links with same-basename replacement files in the selected version are valid updates, regardless of their current target directory or link chain. If a gateway references an obsolete link with no replacement, the entire update fails without changing or removing any links. The error lists the affected links and gateway names; update those gateway setups to remove or replace the obsolete include references, then retry. Disabled and optional includes also protect obsolete links. Unreadable setups or unsafe link resolution also fail the preflight.

After a successful preflight, replacement links are updated and unused obsolete links for the selected source are removed. Other sources' links are left alone. References directly into older version directories do not block link cleanup, since those directories and regular files are not removed. Link changes are sequential, not atomic; completed changes are not rolled back if a later filesystem operation fails.

Version values must be valid source-prefixed or legacy semantic versions. This prevents a version argument from selecting a path outside the includes directory.

## Usage

```text
geneos includes update [flags]
```

### Options

```text
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/geneos/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --prefix string             Select the include source prefix (default "itrs")
      --profiles string           profiles configuration (defaults are ~/.config/geneos/profiles.yaml, /etc/geneos/profiles.yaml)
  -V, --version string            Select a semantic version or full source-version directory
```

## SEE ALSO

* [geneos includes](geneos_includes.md)	 - Manage includes
