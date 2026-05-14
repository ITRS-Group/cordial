# `dv2email`

Email a Dataview following Geneos Action/Effect conventions.

When called without a sub-command and no arguments the program processes environment variables setup as per Geneos Action/Effect conventions and constructs an HTML Email of the dataview the data item is from.

Settings for the Gateway REST connection and defaults for the EMail gateway can be located in dv2email.yaml - either in the working directory or in the user's `.config/dv2email` directory

## Usage

```text
dv2email
```

## Commands

| Command | Description |
|-------|-------|
| [`dv2email export`](dv2email_export.md)	 | Export dataview(s) to local files |

### Options

```text
  -i, --inline-css          inline CSS for better mail client support (default true)
  -t, --to "TO, ..."        "TO, ..." recipients as a comma-separated list of email addresses
  -c, --cc "CC, ..."        "CC, ..." recipients as a comma-separated list of email addresses
  -b, --bcc "BCC, ..."      "BCC, ..." recipients as a comma-separated list of email addresses
  -s, --subject "SUBJECT"   "SUBJECT" of the email
  -f, --config string       config file (default is $HOME/.config/geneos/dv2email.yaml)
  -D, --dataview string     dataview name, ignored if _VARIBLEPATH set in environment
  -E, --entity string       entity name, ignored if _VARIBLEPATH set in environment
  -S, --sampler string      sampler name, ignored if _VARIBLEPATH set in environment
  -T, --type string         type name, ignored if _VARIBLEPATH set in environment
                            To explicitly select empty/no type use --type/-T ""
```

## SEE ALSO

