# geneos ls

List instances

```text
geneos ls [flags] [TYPE] [NAME...]
```

## Details

Matching instances are listed with details.

The default output is intended for human viewing but can be in CSV
format using the `-c` flag or JSON with the `-j` or `-i` flags, the
latter "pretty" formatting the output over multiple, indented lines.

### Options

```text
  -c, --csv      Output CSV
  -j, --json     Output JSON
  -i, --pretty   Output indented JSON
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
