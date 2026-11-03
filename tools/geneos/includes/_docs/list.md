List include files found in the selected Geneos installation and include references used by its Gateways. The default view groups shared and versioned includes by file name. Use one of `--include`, `--version`, or `--gateway` to select a view.

The include view reports copies, Gateway usage, status flags, modification time, size, and location. Files with duplicate basenames are shown as separate numbered rows beneath a summary row. `--all` also includes Gateway-local files.

The version view lists the current include links and available semantic-version directories. Use `--all` to list each file inside those directories.

The Gateway view lists includes configured for each Gateway, ordered by priority. Nested include group names are included with the file name. Gateway-local includes are omitted unless `--all` is set.

Use `--long` for version, permissions, and owner columns where available. Owner information is not currently collected. The `--format` option accepts `column`, `table`, `csv`, `json`, `pretty`, or `toolkit`; column output uses human-readable sizes, while machine-readable formats use byte counts.

Set `--format json` to output the selected view as a compact array of objects, with lower-camel-case field names corresponding to the selected columns. Empty fields are omitted, as is `owner` because owner data is not collected. Counts, priorities, and byte sizes are JSON numbers. Unknown or missing sizes are `-1`; a known empty file has size `0`. Set `--format pretty` for the same JSON output with indentation.

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