The `snapshot` command fetches one or more dataviews using the Geneos Gateway REST Commands API. The TYPE, if given, must be `gateway`. One Gateway instance can be selected by name, or `--connect`/`-c` can select a URL or `HOST:PORT` directly without a configured Gateway. A snapshot execution can connect to at most one Gateway.

XPath names, managed-entity attributes, sampler types, and cell columns are parsed structurally. Other predicates are retained on their path component and sent to the Gateway unchanged, for example `//dataview[(param("Group")="Gateway Info")]`; the Gateway evaluates those predicates.

A `--connect` URL may include a path to replace the default `/rest` Commands API base path, for example `https://gateway.example/proxy/geneos`. An explicit `http://` or `https://` URL selects that connection mode. For `HOST:PORT`, snapshot tries HTTPS first and falls back to HTTP only if the server does not support TLS. TLS certificate-validation errors are not treated as an HTTP fallback.

Authentication to the Gateway is through a combination of command line flags, environment variables, and configuration parameters. Command line credentials take precedence over `ITRS_SNAPSHOT_USERNAME` and `ITRS_SNAPSHOT_PASSWORD`, which take precedence over global `snapshot.username` and `snapshot.password` values. Per-Gateway values take precedence over those defaults, followed by saved credentials when no username is otherwise available. The `--password`/`-p` flag and `ITRS_SNAPSHOT_PASSWORD` accept plaintext or `${enc:...}` values expanded by the configuration package. The user is only prompted for a password when a username is available but no password was found. As the Gateway may be configured to not require authentication, the absence of a username and password is valid and you will not be prompted for a username if one is not found.

For example, set the snapshot username for a Gateway with:

```text
geneos set gateway GATEWAY snapshot.username=alice
```

The default output is in JSON format as an array of dataviews, where each dataview is in the format defined in the Gateway documentation at

<https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_commands_tr.html#fetch_dataviews>

Flags to select which properties of data items are available: `-V`, `-S`, `-Z`, `-U` for value, severity, snooze and user-assignment respectively. If none is given then the default is to fetch values only.

To output the results in a form suitable for consumption by a TOOLKIT sampler, use the `--format toolkit` option, which will return the first matching dataview in the special CSV format used by the TOOLKIT sampler.

For newer Netprobes that support multiple dataviews from TOOLKIT scripts, use the `--format toolkit-multi` option, which will return all matching dataviews in the special CSV format used by the TOOLKIT sampler. You must set the fixed number of dataviews expected in the TOOLKIT sampler configuration, which should be matched to the number given to the `--limit`/`-l` option.

The toolkit output formats only support cell/headline values and cannot be used to set the severity, snooze or user-assignment properties of the returned dataviews and so the options that control those are ignored.

In Toolkit and CSV output, common control characters are rendered as readable escapes such as `\n`, `\r`, and `\t`, backslashes are doubled, and other control characters are discarded.

Toolkit output includes a `samplingStatus` headline. Its value is `OK` on success, or the command error text when sampling fails.

To help capture diagnostic information the `-x` option can be used to capture matching xpaths without the dataview contents. `-l` can be used to limit the number of dataviews (or xpaths) but the limit is not applied in any defined order.
