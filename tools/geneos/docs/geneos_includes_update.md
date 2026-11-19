# `geneos includes update`

Update the top-level include symlinks to point to files in a versioned include directory on the selected host.

By default, the links are updated to `itrs-<version>` embedded in the running `geneos` executable. Use `--version` to select a full directory name already present under `gateway/includes`, such as `itrs-v1.31.0` or `acme-ops-v2.0.0-rc.1`. The prefix is the include **source**; its final version suffix is a `v`-prefixed semantic version, including optional prerelease and build metadata. Legacy unprefixed semantic-version directories remain selectable and are treated as the `itrs` source. If the selected directory is missing, the command returns an error; only the current source-prefixed embedded version is extracted automatically when needed.

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
      --profiles string           profiles configuration (defaults are ~/.config/geneos/profiles.yaml, /etc/geneos/profiles.yaml)
  -V, --version string            Specify full source-version directory to update includes to
```

## SEE ALSO

* [geneos includes](geneos_includes.md)	 - Manage includes
