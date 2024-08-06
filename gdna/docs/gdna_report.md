# `gdna report`

The `gdna report` command runs the configured reports and publishes them to the configured Netprobe to be displayed in the Geneos Active Console.

Command line flags allow you to control the destination as well as the format and which reports to run.

You can select the format of the reports using `--format`/`-F`. The supported formats are:

| Format     | Description                                                                                                     |
| ---------- | --------------------------------------------------------------------------------------------------------------- |
| `dataview` | The reports are published as Dataviews to the configured Netprobe                                               |
| `table`    | The reports are formatted as human readable tables                                                              |
| `html`     | The reports are formatted as individual HTML tables (not a complete web page)                                   |
| `toolkit`  | The first matching report is published as a customised CSV suitable for consumption by a Geneos Toolkit sampler |
| `xlsx`     | The reports are published as sheets in an XLSX workbook                                                         |

For all formats except `dataview` you can save the output to a file using the `--output`/`-o` option. The default (`-`) is to write the output to the console on STDOUT, including `xlsx` workbooks.

For the `dataview` format, the Netprobe connection details can be overridden from those in the configuration file using the `--hostname`/`-H`, `--port`/`-P`, `--tls`/`-T`, `--skip-verify`/`-k` options as well as overriding the Managed Entity using `--entity`/`-e` and the Sampler using `--sampler`/`-s` options. When publishing in `dataview` format the `--reset`/`-R` flag can be used to reset any existing Dataview with the same name, which shiould be used when developing reports and the column details changing.

Some reports may contain information considered sensitive, such as server names, host IDs, MAC addresses etc. These can be opaqued in reports using the `--scramble`/`-S` flag.
```text
gdna report
```

### Options

```text
  -o, --output file         output destination file, default is console (default "-")
  -F, --format format       output format (dataview, table, html, toolkit, xslx) (default "dataview")
  -r, --reports string      Run only matching (file globbing style) reports
  -S, --scramble            Scramble configured column of data in reports with sensitive data
  -H, --hostname hostname   Connect to netprobe at hostname (default "localhost")
  -P, --port port           Connect to netprobe on port (default 7036)
  -T, --tls                 Use TLS connection to Netprobe
  -k, --skip-verify         Skip certificate verification for Netprobe connections
  -e, --entity Entity       Send reports to Managed Entity (default "GDNA")
  -s, --sampler Sampler     Send reports to Sampler (default "GDNA")
  -R, --reset               Reset/Delete configured Dataviews
```

## SEE ALSO

* [gdna](gdna.md)	 - Process Geneos License Usage Data
