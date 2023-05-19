# `geneos list`

List instances

```text
geneos list [flags] [TYPE] [NAME...]
```

List details of matching instances.

As for other commands if a `TYPE` is not given all `TYPE`s are included
and if no `NAME` is given all instances for `TYPE` are included. Unless
`NAME` is given in the format `NAME@HOST` then instances from all hosts
are considered. The host can also be controlled using the `--host`/`-H`
global option.

The default output is a table format intended for humans but this can be
changed to CSV format using the `--csv`/`-c` flag or JSON with the
`--json`/`-j` or `--pretty`/`-i` options, the latter option formatting
the output over multiple, indented lines.

In plain output format (i.e. not CSV or JSON) the instance name may be
tagged with a '*' or a '+' which indicate disabled and protected,
respectively.

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

* [geneos](geneos.md)	 - Take control of your Geneos environments
