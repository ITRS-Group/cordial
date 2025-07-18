# `geneos list`

List details of matching instances.

As for other commands if a `TYPE` is not given all `TYPE`s are included and if no `NAME` is given all instances for `TYPE` are included. Unless `NAME` is given in the format `NAME@HOST` then instances from all hosts are considered. The host can also be controlled using the `--host`/`-H` global option.

The default output is a table format intended for humans but this can be changed to CSV format using the `--csv`/`-c` flag or JSON with the `--json`/`-j` or `--pretty`/`-i` options, the latter option formatting the output over multiple, indented lines. There is also a `--toolkit/-t` output format which produces CSV formatted data suitable for consumption by a Geneos Toolkit plugin.

In plain, table output, format the Flags column contains:

  * `A` - Auto Start
  * `D` - Disabled
  * `P` - Protected
  * `R` - Running
  * `T` - TLS enabled

In other output formats each flag gets it's own column or field.

```text
geneos list [flags] [TYPE] [NAME...]
```

### Options

```text
  -c, --csv       Output CSV
  -j, --json      Output JSON
  -i, --pretty    Output indented JSON
  -t, --toolkit   Output Toolkit formatted CSV
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
