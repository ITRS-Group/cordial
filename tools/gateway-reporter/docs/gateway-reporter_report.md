# `gateway-reporter report`

Report on Geneos Gateway XML files

```text
gateway-reporter report [flags] [SETUP...]
```

The `report` command will produce report(s) for the Gateway setup files given as arguments. With no arguments the command will read a configuration file from STDIN, which can also be given as `-`. Paths to setup files can be local files, including prefixed with a `~/` to indicate relative to the users home directory or a URL to a remote setup file.

The contents of the reports, their formats and destination directory are taken from command configuration files and built-in defaults. Configuration files are loaded from the following locations, each one found overriding similar settings from the previous one (and the defaults):

* `/etc/itrs/gateway-reporter.yaml`
* `${HOME}/.config/geneos/gateway-reporter.yaml`
* `./gateway-reporter.yaml`

For each Gateway setup file given one or more reports are written to the configured directory. The names of the default report file are determined by the configuration file. The defaults are:

* XLSX - An XLSX file containing multiple sheets for each report - `${gateway}-Report-${datetime}.xlsx`
* CSV - A ZIP file of CSV files for each report - `${gateway}-Report-${datetime}.zip`
* JSON - A JSON formatted report for processing - `${gateway}-Report-${datetime}.json`
* XML - The merged XML setup file for diagnostics and audit - `${gateway}-Merged-${datetime}.xml`

The `${gateway}` and `${datetime}` are replaced with the Gateway name (or prefix, see below) and the date and time of the start of the report creation (`YYYYMMDDhhmmss`).

The output directory can be set with the `--output/-o DIR` option. Merging is controlled by the `--merge/-m` option, which requires an installed Gateway package given with the `--install` option - this can either be to the package directory or the Gateway binary - but in the other files from the installation must be present in the same location.

If the setup file being processed does not contain a Gateway name, either because the setup file is not bering fully merged or the Gateway name is set on the command line, then use the `--prefix/-p` option to set the `${gateway}` in the report paths above and in the reports themselves.

### Options

```text
  -o, --out DIRECTORY        Write reports to DIRECTORY. Default `/tmp/gateway-reporter`
  -m, --merge                Create a merged config file. --install must be set
  -i, --install BINARY|DIR   Path to the gateway installation BINARY|DIR
  -p, --prefix name          Report prefix for configurations without a Gateway name
```

### Options inherited from parent commands

```text
  -f, --config string   config file (default is $HOME/.config/geneos/docs.yaml)
  -d, --debug           enable extra debug output
```

## SEE ALSO

* [gateway-reporter](gateway-reporter.md)	 - Report on Geneos Gateway XML files
