# `gdna list reports`

`gdna list` displays a list of configured reports along with information about the report title and any restrictions on running for certain types of outputs.

The output is formatted as a text table by default, for display on the console. You can also select alternative formats using the `--format`/`-F` flag. Supported formats are `table` (the default), `html` for a HTML table, `toolkit`/`csv` for CSV suitable for a Toolkit sampler, `markdown`/`md` or `tsv`. If the format is not recognised then the default `table` format is used.

You can limit the list of reports to list using the `--report`/`-r` flag, which supports glob=style wildcards.

## Usage

```text
gdna list reports
```

### Options

```text
  -r, --report string   Run only the matching reports, for multiple reports use a
                        comma-separated list. Report names can include shell-style wildcards.
                        Split reports can be suffixed with ':value' to limit the report
                        to the value given.
  -F, --format string   format output. supported formats: 'html', 'table', 'tsv', 'toolkit', 'markdown' (default "table")
  -f, --config FILE     Use configuration FILE
  -l, --logfile file    Write logs to file. Use '-' for console or /dev/null for none (default "docs.log")
```

## SEE ALSO

* [gdna list](gdna_list.md)	 - List commands
