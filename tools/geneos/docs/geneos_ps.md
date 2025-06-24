# `geneos ps`

The `ps` command will report details of matching and running instances. It can also report on open files and sockets.

The `--long/-l` flag alsop reports a number of other metrics, largely derived from the `/proc` pseudo-filesystem. Also, as it potentially takes significant time to lookup ports for remote instances these are not shown by default. Use the `--long`/`-l` option to see these.

To see open files use the `--files/-f` flag, or to see open sockets use the `--network/-n` flag. The meaning of the columns will be documented later. The previous meaning of the short `-n` flag has changed and the `--nolookup` option was removed.

The default output is a table format intended for humans but this can be changed to CSV format using the `--csv`/`-c` flag or JSON with the `--json`/`-j` or `--pretty`/`-i` options, the latter option formatting the output over multiple, indented lines. Use the `--toolkit/-t` flag to report in Geneos Toolkit specific CSV format, which includes headlines and a unique first column to act as the row name.

```text
geneos ps [flags] [TYPE] [NAMES...]
```

### Options

```text
  -n, --network   Show TCP sockets
  -l, --long      Show more output (remote ports etc.)
  -j, --json      Output JSON
  -i, --pretty    Output indented JSON
  -c, --csv       Output CSV
  -t, --toolkit   Output Toolkit formatted CSV
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
