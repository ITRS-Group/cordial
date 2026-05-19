The `gdna explain` command is for users writing new reports and want to see how the report query will be evaluated by the SQLite query analyser. The command only works for "basic" reports, i.e. those in the output of `gdna list` with and empty value in the TYPE column.

The output shows the underlying SQL query, with all embedded parameters resolved and then a tree style `EXPLAIN` query output.

Instead of explaining the query used for a named report, using the `--expand-query`/`-q` flag will show the expanded query with all configuration variables resolved. The path to the query must include all configuration levels, e.g. `db.unused-gateways.create`. This is useful for diagnostics when the query is not working as expected.
