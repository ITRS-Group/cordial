List include files found in the selected Geneos installation and include references used by its Gateways. The default view groups shared and versioned includes by file name. Use one of `--include`, `--version`, or `--gateway` to select a view.

The include view reports copies, Gateway usage, status flags, modification time, size, and location. Files with duplicate basenames are shown as separate numbered rows beneath a summary row. `--all` also includes Gateway-local files.

The version view lists the current include links and available semantic-version directories. Use `--all` to list each file inside those directories.

The Gateway view lists includes configured for each Gateway, ordered by priority. The `File` field contains only the include file's basename; nested group names do not change it. Gateway-local includes are omitted unless `--all` is set.

With `--gateway`, optional instance names or shell-style glob patterns limit output to matching Gateways. Patterns support `*`, `?`, and character classes such as `[ab]`; quote them to prevent shell expansion. Multiple names or patterns are combined without duplicate rows. With no names, all Gateways are listed; patterns matching no Gateways produce no rows. Name filtering applies only to the Gateway view.

```shell
geneos includes list --gateway 'prod-*'
geneos includes list --gateway gateway-a gateway-b
```

Use `--long` for version, permissions, and owner columns where available. Owner information is not currently collected. The `--format` option accepts `column`, `table`, `csv`, `json`, `pretty`, or `toolkit`; column output uses human-readable sizes, while machine-readable formats use byte counts.

Set `--format json` to output the selected view as a compact array of objects using the long fields regardless of `--long`. Permission values use octal strings such as `"0644"`; table output retains symbolic permissions. Fields use lower-camel-case names. Empty fields are omitted, as is `owner` because owner data is not collected. Counts, priorities, and byte sizes are JSON numbers. Unknown or missing sizes are `-1`; a known empty file has size `0`. Set `--format pretty` for the same JSON output with indentation.

In JSON output, the aggregate `flags` field is replaced by optional boolean fields: `disabled` (`D`), `local` (`L`), `readOnly` (`R`), `shared` (`S`), `remote` (`U`), `missing` (`X`), and `optional` (`o`). A field is included with value `true` only when that flag applies; false fields are omitted. Versioned includes are identified by their non-empty `version` field, so JSON does not repeat the `V` flag.

Flags identify include properties:

* `D` - Disabled in the Gateway setup
* `L` - Gateway-local include
* `o` - Optional include
* `R` - Read-only file
* `S` - Shared include
* `U` - HTTP or HTTPS URL include
* `V` - Versioned include
* `X` - Local file is missing

Remote locations are queried with HTTP HEAD requests for modification time and size unless `--no-remote` is set. Failures to retrieve remote metadata do not prevent listing the configured location.