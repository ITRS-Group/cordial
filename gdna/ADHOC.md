# GDNA _Ad-Hoc_ Reporting

GDNA supports ad-hoc reporting to allow you to quickly generate reports on demand without needing to create and manage configuration files, a local SQLite database, or report definitions.

To create ad-hoc reports use the `gdna report --adhoc` command along with the relevant options to specify the report type and any filters you wish to apply. When combined with the `--zip`/`-Z` option the report output will be a ZIP file containing the report data in CSV format.

GDNA is also available as a Windows binary, `gdna.exe`, which can be run from a command prompt or PowerShell window. This allows you to run ad-hoc reports directly from a Windows system without needing to install Go or build the program from source.

Without a `gdna.yaml` configuration file you may need to override built in defaults. These can be specified using environment variables that start `GDNA_` and then the configuration file option, with dots replaced by underscores and in all upper case. For example, to skip TLS certificate verification, which is normally `gdna.licd-skip-verify`, you would set the `GDNA_GDNA_LICD_SKIP_VERIFY` environment variable to `true`. Note that the double `GDNA_` prefix is required as the first one indicates it is an environment variable for GDNA and the second one is part of the configuration option name.

When running ad-hoc reports, **all** logging is disabled by default. To see log output use the `--log`/`-l` option with the destination file or `-` for standard output.

## Examples

>[!NOTE]
>To see a list of available reports, run `gdna list reports`

### Linux

To see a CSV formatted report of all servers, from the license daemon running locally and skipping TLS certificate verification:

```bash
export GDNA_GDNA_LICD_SKIP_VERIFY=true
gdna report --adhoc --reports servers
```

or

```bash
GDNA_GDNA_LICD_SKIP_VERIFY=true gdna report --adhoc --reports servers
```

To create a ZIP file with all reports, connecting to a license daemon running on `licd.example.com` at port `7041`:

```bash
gdna report --adhoc --zip --source https://licd.example.com:7041 --output report.zip
```

or an XLSX workbook for multiple license daemons, and see the log output on standard output including errors:

```bash
gdna report --adhoc --format xlsx --source https://licd.example.com:7041 \
  --source https://licd2.example.com:7041 --output report.xlsx --log -
```

### Windows Powershell

To see a CSV formatted report of all servers, from the license daemon running locally and skipping TLS certificate verification:

```powershell
$Env:GDNA_GDNA_LICD_SKIP_VERIFY="true"
gdna.exe report --adhoc --reports servers
```

To create a ZIP file with all reports in CSV format, connecting to a license daemon running on `licd.example.com` at port `7041`:

```powershell
gdna.exe report --adhoc --zip --source https://licd.example.com:7041 --output report.zip
```

or an XLSX workbook for multiple license daemons, and see the log output on standard output including errors:

```powershell
gdna.exe report --adhoc --format xlsx --source https://licd.example.com:7041 `
  --source https://licd2.example.com:7041 --output report.xlsx --log -
```
