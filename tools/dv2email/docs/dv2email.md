# `dv2email`

Email a Dataview following Geneos Action/Effect conventions

```text
dv2email [flags]
```
Email a Dataview following Geneos Action/Effect conventions.

When called without a sub-command and no arguments the program processes environment variables setup as per Geneos Action/Effect conventions and constructs an HTML Email of the dataview the data item is from.

Settings for the Gateway REST connection and defaults for the EMail gateway can be located in dv2email.yaml (either in the working directory or in the user's .config/dv2email directory)

### Options

```text
  -E, --entity string     entity name
  -S, --sampler string    sampler name
  -T, --type string       type name
  -D, --dataview string   dataview name
  -t, --to string         To as comma-separated emails
  -c, --cc string         Cc as comma-separated emails
  -b, --bcc string        Bcc as comma-separated emails
  -f, --config string     config file (default is $HOME/.config/geneos/dv2email.yaml)
  -d, --debug             enable extra debug output
  -i, --inline-css        inline CSS for better mail client support (default true)
```

