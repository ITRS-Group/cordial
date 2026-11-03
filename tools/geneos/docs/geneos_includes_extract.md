# `geneos includes extract`

Extract the include files embedded in the `geneos` executable to the selected host.

Managed include files are written under `gateway/includes/<version>`. Existing files that differ are moved to timestamped `.bak` files before replacement. Unchanged files are left in place. If the version directory already exists, extraction requires `--force`.

Configuration templates are written under `gateway/includes/config`. Existing configuration templates are never overwritten because they may contain local changes.

Use `--update` to update the top-level links to the extracted version after extraction. `--force` also allows those links to be replaced. If extraction or a filesystem operation fails, the command returns an error; successfully completed earlier operations are not rolled back.

## Usage

```text
geneos includes extract [flags]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -F, --force             Overwrite existing files
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
  -U, --update            Update version links for existing include files
```

## SEE ALSO

* [geneos includes](geneos_includes.md)	 - Manage includes
