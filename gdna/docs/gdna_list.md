# `gdna list`

`gdna list` displays a list of configured reports along with information about the report title and any restrictions on running for certain types of outputs.

The output is formatted as a text table by default, for display on the console. You can also select alternative formats using the `--format`/`-F` flag. Supported formats are `table` (the default), `html` for a HTML table, `toolkit`/`csv` for CSV suitable for a Toolkit sampler, `markdown`/`md` or `tsv`. If the format is not recognised then the default `table` format is used.

You can limit the list of reports to list using the `--report`/`-r` flag, which supports glob=style wildcards.


## Commands

| Command / Aliases | Description |
|-------|-------|
| [`gdna list excludes / exclude`](gdna_list_excludes.md)	 | List excluded items |
| [`gdna list groups / group / grouping / groupings`](gdna_list_groups.md)	 | List groups |
| [`gdna list includes / include`](gdna_list_includes.md)	 | List excluded items |
| [`gdna list reports / report`](gdna_list_reports.md)	 | List available reports |

### Options

```text
  -f, --config FILE     Use configuration FILE
  -F, --format string   format output. supported formats: 'html', 'table', 'tsv', 'toolkit', 'markdown' (default "table")
  -l, --logfile file    Write logs to file. Use '-' for console or /dev/null for none (default "docs.log")
```

## SEE ALSO

* [gdna](gdna.md)	 - Process Geneos License Usage Data
