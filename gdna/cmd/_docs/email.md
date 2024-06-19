# `gdna email`

The `gdna email` command runs reports and sends the results to the configured email destinations.

You can limit the reports included using the `--report`/`-r` option, which accepts a single parameter which can be an individual report name of a glob-style wildcard that may match multiple reports. Remember to quote any special characters to avoid shell expansion. To see which reports are available use the `gdna list` command.

You can override some of the configuration file settings using command ling flags to set the email Subject with the `--subject` flag, the sender From address with the `--from` flag and the recipients using `--to`/`--cc`/`--bcc` flags.
