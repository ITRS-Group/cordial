# `geneos includes update`

Update the top-level include symlinks to point to files in a versioned include directory on the selected host.

By default, the links are updated to the version embedded in the running `geneos` executable. Use `--version` to select another semantic version already present under `gateway/includes`. If the selected version directory is missing, the command returns an error; the current embedded version is extracted automatically when needed.

Links that already point to the selected version are left unchanged. If a target path already exists with a different link or file, it is removed before creating the new symlink.

Before changing any links, a preflight checks obsolete top-level symlinks using the same gateway setup loader as `list --gateway`, including nested include groups. Links with same-basename replacement files in the selected version are valid updates, regardless of their current target directory or link chain. If a gateway references an obsolete link with no replacement, the entire update fails without changing or removing any links. The error lists the affected links and gateway names; update those gateway setups to remove or replace the obsolete include references, then retry. Disabled and optional includes also protect obsolete links. Unreadable setups or unsafe link resolution also fail the preflight.

After a successful preflight, replacement links are updated and unused obsolete links are removed. References directly into older version directories do not block link cleanup, since those directories and regular files are not removed. Link changes are sequential, not atomic; completed changes are not rolled back if a later filesystem operation fails.

Version values must be valid semantic versions. This prevents a version argument from selecting a path outside the includes directory.
## Usage

```text
geneos includes update [flags]
```

### Options

```text
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string           profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
  -V, --version string            Specify version to update includes to
```

## SEE ALSO

* [geneos includes](geneos_includes.md)	 - Manage includes
