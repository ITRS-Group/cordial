# `gdna email`

The `gdna email` command runs reports and sends the results to the configured email destinations.

You can limit the reports included using the `--report`/`-r` option, which accepts a single parameter which can be an individual report name of a glob-style wildcard that may match multiple reports. Remember to quote any special characters to avoid shell expansion. To see which reports are available use the `gdna list` command.

You can override some of the configuration file settings using command ling flags to set the email Subject with the `--subject` flag, the sender From address with the `--from` flag and the recipients using `--to`/`--cc`/`--bcc` flags.

```text
gdna email
```

### Options

```text
  -r, --report string     Run only the matching reports, for multiple reports use a
                          comma-separated list. Report names can include shell-style wildcards.
                          Split reports can be suffixed with ':value' to limit the report
                          to the value given.
      --contents string   Override configured email contents
      --subject string    Override configured email Subject
      --from string       Override configured email From
      --to string         Override configured email To
                          (comma separated, but remember to quote as one argument)
      --cc string         Override configured email Cc
                          (comma separated, but remember to quote as one argument)
      --bcc string        Override configured email Bcc
                          (comma separated, but remember to quote as one argument)
```

## SEE ALSO

* [gdna](gdna.md)	 - Process Geneos License Usage Data
