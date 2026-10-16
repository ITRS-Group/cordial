# `geneos host list`

List the matching remote hosts.

## Usage

```text
geneos host list [flags] [NAME...]
```

### Options

```text
  -a, --all               Show all hosts
  -j, --json              Output JSON
  -i, --pretty            Output indented JSON
  -c, --csv               Output CSV
  -t, --toolkit           Output Toolkit formatted CSV
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos host](geneos_host.md)	 - Remote Host Operations
