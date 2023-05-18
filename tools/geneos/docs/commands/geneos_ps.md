# geneos ps

List process information for instances, optionally in CSV or JSON format

```text
geneos ps [flags] [TYPE] [NAMES...]
```

## Details

Show the status of the matching instances.

### Options

```text
  -f, --files    Show open files
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

* [geneos](geneos.md)	 - Control your Geneos environment
