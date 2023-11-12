When run with no arguments `gateway-reporter` will work as a Gateway validation hook, using the configuration defaults and those in external `gateway-reporter.yaml` files, to produce monitoring coverage reports based on the merged configuration supplied by the executing Gateway. There must be a symbolic link from `validate-setup` in the Gateway hooks directory to the `gateway-reporter` program. If you choose to rename the program to `validate-setup` and place it in the configured hooks directory then all configuration files will also have to be renamed to `validate-setup.yaml` to match.

The contents of the reports, their formats and destination directory are taken from configuration files and built-in defaults. The configuration file is normally loaded from the following locations, each one found overriding similar settings from the previous one (and the defaults):

* `/etc/itrs/gateway-reporter.yaml`
* `${HOME}/.config/geneos/gateway-reporter.yaml`
* `./gateway-reporter.yaml`

Note that the working directory for the program in validation hook mode is the same as the Gateway invoking it, allowing each gateway to have it's own configuration.

The names of the default report file are determined by the configuration file. The defaults are:

* XLSX - An XLSX file containing multiple sheets for each report - `${gateway}-Report-${datetime}.xlsx`
* CSV - A ZIP file of CSV files for each report - `${gateway}-Report-${datetime}.zip`
* JSON - A JSON formatted report for processing - `${gateway}-Report-${datetime}.json`
* XML - The merged XML setup file for diagnostics and audit - `${gateway}-Merged-${datetime}.xml`

The `${gateway}` and `${datetime}` are replaced with the Gateway name (or prefix, see below) and the date and time of the start of the report creation (`YYYYMMDDhhmmss`).


If the Gateway is permanently configured with a hooks directory then the `gateway-reporter` will be run every time the setup is validated or saved or when the Gateway starts. To reduce the number of reports the program silently exits, reporting no errors, except for "Validate" or "Command" invocations. See the Gateway documentation for when these events occur.

The other way to run `gateway-reporter` as a validation hook is to use `geneos show -V --hook DIR`, which runs a single-shot Gateway with the appropriate arguments to invoke the validation hook.

To have more control over the Gateway configuration merging process or to use a pre-merged file, use the `gateway-reporter report` command.
