# `geneos logs`

View, search or follow logs

```text
geneos logs [flags] [TYPE] [NAME...]
```

The default behaviour is to show the last 10 lines of the log file for
each matching instance. The order of instances cannot be predicted.

You can control the basic behaviour of the command with three options.
The `--lines`/`-n` option controls how many lines to output per instance
at the start of the program. The `--cat`/`-c` options will output the
whole log file and any `--lines`/`-n` option is ignored. The
`--follow`/`-f` option will show the last 10 lines (unless you ask for
more with the `--lines`/`-n` option) and then wait for the log to be
updated, just like the standard `tail -f` command except it will work
for all matching instances including remote ones. `--cat`/`-c` and
`--follow`/`-f` are mutually exclusive.

The `--stderr`/`-E` option controls whether the separate `STDERR` log
(if there is one) for each matching instance is also shown along with
the main log. There is no way currently to only view the `STDERR` logs.

The `--match`/`-g` and `--ignore`/`-v` options will filter lines the
output based on a case sensitive search over the whole line. As can be
expected `--match`/`-g` behaves somewhat like `grep` and `--ignore`/`-v`
like `grep -v`. Case-insensitive filtering is avoided for performance.

Only on `--match`/`-g` or `--ignore`/`-v` is allowed.

Each block of output has a header indicating the details of the instance
and the path to the log file. The header is output each time the file
being output changes. There is no way to suppress this header.

Future releases may add support for more complex filtering using regular
expressions and also multiple filters.

### Options

```text
  -f, --follow          Follow file
  -n, --lines int       Lines to tail (default 10)
  -c, --cat             Output whole file
  -E, --stderr          Also show STDERR output files
  -g, --match string    Match lines with STRING
  -v, --ignore string   Match lines without STRING
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
