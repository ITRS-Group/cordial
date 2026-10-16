# `geneos host show`

Show details of remote host configurations. If no names are supplied then all configured hosts are shown.

The output is always unprocessed, and so any values in `expandable` format are left as-is. This protects, for example, SSH passwords from being accidentally shown in clear text.

## Usage

```text
geneos host show [flags] [NAME...]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos host](geneos_host.md)	 - Remote Host Operations
