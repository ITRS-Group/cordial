The `ps` command will report details of matching and running instances.

As it potentially takes significant time to lookup ports for remote instances these are not shown by default. Use the `--long`/`-l` option to see these.

In some cases the user and group names may take a while to lookup, not make sense for remote instances or you want to see the underlying UID/GID for processes, in which case you can use the `--nolookup`/`-n` option.

The default output is a table format intended for humans but this can be changed to CSV format using the `--csv`/`-c` flag or JSON with the `--json`/`-j` or `--pretty`/`-i` options, the latter option formatting the output over multiple, indented lines.
