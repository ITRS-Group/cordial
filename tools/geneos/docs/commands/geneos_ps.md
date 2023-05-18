# `geneos ps`

Show running instances

```text
geneos ps [flags] [TYPE] [NAMES...]
```

## Details

The `ps` command will report details of matching and running instances.

The default output is a table format intended for humans but this can
be changed to CSV format using the `--csv`/`-c` flag or JSON with the
`--json`/`-j` or `--pretty`/`-i` options, the latter option
formatting the output over multiple, indented lines.

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

* [geneos](geneos.md)	 - Control your Geneos environment
