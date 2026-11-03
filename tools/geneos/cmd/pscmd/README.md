The `ps` command, which can also be invoked using the alias `status`, will report details of running instances which match any TYPE and/or NAMES given on the command line. It can also report on open files and sockets.

The `--long/-l` flag also reports a number of other metrics, largely derived from the Linux `/proc` filesystem. Also, as it potentially takes significant time to lookup ports for remote instances these are not shown by default. Use the `--long`/`-l` option to see these.

Use `--files`/`-f` to list open files or `--network`/`-n` to list TCP sockets. These flags select different output views and cannot be used together.

The default output is a table format intended for humans but this can be changed to CSV format using the `--csv`/`-c` flag or JSON with the `--json`/`-j` or `--pretty`/`-i` options, the latter option formatting the output over multiple, indented lines. Use the `--toolkit/-t` flag to report in Geneos Toolkit specific CSV format, which includes headlines and a unique first column to act as the row name.

Any Netprobes that have a running Collection Agent managed process will show a `netprobe/ca` entry along with details of the Collection Agent process.