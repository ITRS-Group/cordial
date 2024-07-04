`gdna list` displays a list of configured reports along with information about the report title and any restrictions on running for certain types of outputs.

The output is formatted as a text table by default, for display on the console. You can also select alternative formats using the `--format`/`-F` flag. Supported formats are `table` (the default), `html` for a HTML table, `toolkit`/`csv` for CSV suitable for a Toolkit sampler, `markdown`/`md` or `tsv`. If the format is not recognised then the default `table` format is used.

You can limit the list of reports to list using the `--report`/`-r` flag, which supports glob=style wildcards.
