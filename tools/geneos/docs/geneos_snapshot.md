# `geneos snapshot`

Capture a snapshot of each matching dataview

```text
geneos snapshot [flags] [gateway] [NAME] XPATH...
```

Snapshot one or more dataviews using the REST Commands API endpoint introduced in GA5.14. The TYPE, if given, must be `gateway`.

Authentication to the Gateway is through a combination of command line flags and configuration parameters. If either of the parameters `snapshot.username` or `snapshot.password` is defined for the Gateway or globally then this is used as a default unless overridden on the command line by the `-u` and `-P` options. The user is only prompted for a password if it cannot be located in either of the previous places.

<!-- CREDENTIALS - also, fix them, gateway:NAME@HOST (if not local) -->

The output is in JSON format as an array of dataviews, where each dataview is in the format defined in the Gateway documentation at

<https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_commands_tr.html#fetch_dataviews>

Flags to select which properties of data items are available: `-V`, `-S`, `-Z`, `-U` for value, severity, snooze and user-assignment respectively. If none is given then the default is to fetch values only.

To help capture diagnostic information the `-x` option can be used to capture matching xpaths without the dataview contents. `-l` can be used to limit the number of dataviews (or xpaths) but the limit is not applied in any defined order.

### Options

```text
  -V, --value             Request cell values (default true)
  -S, --severity          Request cell severities
  -Z, --snooze            Request cell snooze info
  -U, --userassignment    Request cell user assignment info
  -u, --username string   Username
  -P, --pwfile string     Password
  -l, --limit int         limit matching items to display. default is unlimited. results unsorted.
  -x, --xpaths            just show matching xpaths
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
