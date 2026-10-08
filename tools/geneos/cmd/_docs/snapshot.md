The `snapshot` command fetches one or more dataviews using the Geneos Gateway REST Commands API. The TYPE, if given, must be `gateway`. A Gateway instance name must be given or the use the wildcard `all`. Not providing a Gateway name or `all` will result in no data being returned.

Authentication to the Gateway is through a combination of command line flags and configuration parameters. If the parameters `snapshot::username` or `snapshot::password` are defined for the Gateway (see `geneos help gateway`) then these are used as a default unless overridden on the command line by the `--user`/`-u` option. The user is only prompted for a password if it cannot be located in the configuration or in saved credentials. As the Gateway may be configured to not require authentication, the absence of a username and password is valid and you will not be prompted for a username if one is not found.

The default output is in JSON format as an array of dataviews, where each dataview is in the format defined in the Gateway documentation at

<https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_commands_tr.html#fetch_dataviews>

Flags to select which properties of data items are available: `-V`, `-S`, `-Z`, `-U` for value, severity, snooze and user-assignment respectively. If none is given then the default is to fetch values only.

To output the results in a form suitable for consumption by a TOOLKIT sampler, use the `--format toolkit` option, which will return the first matching dataview in the special CSV format used by the TOOLKIT sampler.

For newer Netprobes that support multiple dataviews from TOOLKIT scripts, use the `--format toolkit-multi` option, which will return all matching dataviews in the special CSV format used by the TOOLKIT sampler. You must set the fixed number of dataviews expected in the TOOLKIT sampler configuration, which should be matched to the number given to the `--limit`/`-l` option.

The toolkit output formats only support cell/headline values and cannot be used to set the severity, snooze or user-assignment properties of the returned dataviews and so the options that control those are ignored.

To help capture diagnostic information the `-x` option can be used to capture matching xpaths without the dataview contents. `-l` can be used to limit the number of dataviews (or xpaths) but the limit is not applied in any defined order.
