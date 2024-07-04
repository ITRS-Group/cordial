The `gdna explain` command is for users writing new reports and want to see how the report query will be evaluated by the SQLite query analyser. The command only works for "basic" reports, i.e. those in the output of `gdna list` with and empty value in the TYPE column.

The output shows the underlying SQL query, with all embedded parameters resolved and then a tree style `EXPLAIN` query output.
