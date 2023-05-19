# `geneos host list`

List hosts, optionally in CSV or JSON format

```text
geneos host list [flags] [TYPE] [NAME...]
```

List the matching remote hosts.

### Options

```text
  -j, --json     Output JSON
  -i, --pretty   Output indented JSON
  -c, --csv      Output CSV
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos host](geneos_host.md)	 - Manage remote host settings
