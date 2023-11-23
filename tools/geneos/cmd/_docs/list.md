List details of matching instances.

As for other commands if a `TYPE` is not given all `TYPE`s are included and if no `NAME` is given all instances for `TYPE` are included. Unless `NAME` is given in the format `NAME@HOST` then instances from all hosts are considered. The host can also be controlled using the `--host`/`-H` global option.

The default output is a table format intended for humans but this can be changed to CSV format using the `--csv`/`-c` flag or JSON with the `--json`/`-j` or `--pretty`/`-i` options, the latter option formatting the output over multiple, indented lines.

In plain (human readable) format the Flags column contains:

  * `D` - Disabled
  * `P` - Protected
  * `A` - Auto Start
  * `T` - TLS enabled (for at last one connection type)

In other output formats each flag gets it's own column or field.
