# `geneos ps`

List Running Instance Details

```text
geneos ps [flags] [TYPE] [NAMES...]
```

The `ps` command will report details of matching and running instances.

The default output is a table format intended for humans but this can be
changed to CSV format using the `--csv`/`-c` flag or JSON with the
`--json`/`-j` or `--pretty`/`-i` options, the latter option formatting
the output over multiple, indented lines.

### Options

```text
  -j, --json     Output JSON
  -i, --pretty   Output indented JSON
  -c, --csv      Output CSV
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
