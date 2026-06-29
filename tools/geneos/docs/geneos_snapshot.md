# `geneos snapshot`

The `snapshot` command fetches one or more dataviews using the Geneos Gateway REST Commands API. The TYPE, if given, must be `gateway`. A Gateway instance name must be given or the use the wildcard `all`. Not providing a Gateway name or `all` will result in no data being returned.

Authentication to the Gateway is through a combination of command line flags and configuration parameters. If the parameters `snapshot::username` or `snapshot::password` are defined for the Gateway (see `geneos help gateway`) then these are used as a default unless overridden on the command line by the `--user`/`-u` option. The user is only prompted for a password if it cannot be located in the configuration or in saved credentials. As the Gateway may be configured to not require authentication, the absence of a username and password is valid and you will not be prompted for a username if one is not found.

The output is in JSON format as an array of dataviews, where each dataview is in the format defined in the Gateway documentation at

<https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_commands_tr.html#fetch_dataviews>

Flags to select which properties of data items are available: `-V`, `-S`, `-Z`, `-U` for value, severity, snooze and user-assignment respectively. If none is given then the default is to fetch values only.

To help capture diagnostic information the `-x` option can be used to capture matching xpaths without the dataview contents. `-l` can be used to limit the number of dataviews (or xpaths) but the limit is not applied in any defined order.

## Usage

```text
geneos snapshot [flags] [gateway] [NAME] XPATH...
```

### Options

```text
  -V, --value             Request cell values (default true)
  -S, --severity          Request cell severities
  -Z, --snooze            Request cell snooze info
  -U, --userassignment    Request cell user assignment info
  -u, --username string   Username
  -l, --limit int         limit matching items to display. default is unlimited. results unsorted.
  -x, --xpaths            just show matching xpaths
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
