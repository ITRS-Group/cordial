# `geneos includes update`

Update the top-level include symlinks to point to files in a versioned include directory on the selected host.

By default, the links are updated to the version embedded in the running `geneos` executable. Use `--version` to select another semantic version already present under `gateway/includes`. If the selected version directory is missing, the command returns an error; the current embedded version is extracted automatically when needed.

Links that already point to the selected version are left unchanged. If a target path already exists with a different link or file, the command returns an error unless `--force` is specified. `--force` removes the existing path before creating the new symlink.

Version values must be valid semantic versions. This prevents a version argument from selecting a path outside the includes directory.
## Usage

```text
geneos includes update [flags]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -F, --force             Overwrite existing files
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
  -V, --version string    Specify version to update includes to
```

## SEE ALSO

* [geneos includes](geneos_includes.md)	 - Manage includes
