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

For `toolkit` and `csv` format reports you must select exactly one report name with the `--reports`/`-r` option. For other formats you can select multiple reports separated by commas, or leave this option out to run all configured reports.

For ad-hoc reports that do not use an on-disk database, use the `--adhoc`/`-A` flag to fetch license usage data from the configured sources, build the in-memory database and then run the reports against that data. This type of reporting can be run without a configuration file, defaulting to the built-in values for configuration items. It is worth noting that you can use environment variables to set configuration values without your own YAML file; e.g. to set a password for an XLSX output file you can set `GDNA_XLSX_PASSWORD` (which corresponds to the `xlsx.password` configuration item), like this:

```bash
GDNA_XLSX_PASSWORD=example gdna report --adhoc --source https://licd:7041 --output report.xlsx --format xlsx
```

Similarly, to skip TLS verification when fetching license data over HTTPS you can set the `gdna.licd-skip-verify` configuration item using the environment variable `GDNA_GDNA_LICD_SKIP_VERIFY` (dashes in configuration items are converted to underscores, but the top-level `gdna` must be included, in addition to the environment variable prefix of `GDNA_`), like this:

```bash
GDNA_GDNA_LICD_SKIP_VERIFY=true gdna report --adhoc --source https://licd:7041 --output report.csv --format csv --report servers
```

When using this option you can also use the `--source`/`-L` flag one or more times to override the configured license data sources and specify which source(s) to use. This is useful for testing new or modified sources without changing the configuration file.

```text
gdna report
```

### Options

```text
  -o, --output file         output destination file, default is console (stdout) (default "-")
  -F, --format format       output format - one of: dataview, table, html, markdown,
                            toolkit, csv, xslx (default "dataview")
  -A, --adhoc               Ad-hoc reporting: Fetch license reports, build data in-memory and report
                            (default format CSV, dataview output not supported)
  -L, --source URL | PATH   Override configured licence source.
                            (Repeat as required)
  -r, --reports string      Run only the matching reports, for multiple reports use a
                            comma-separated list. Report names can include shell-style wildcards.
                            Split reports can be suffixed with ':value' to limit the report
                            to the value given.
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
