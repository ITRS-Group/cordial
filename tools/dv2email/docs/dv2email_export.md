# `dv2email export`

Export dataview(s) to local files

```text
dv2email export [flags]
```

Email a Dataview following Geneos Action/Effect conventions.

When called without a sub-command and no arguments the program processes environment variables setup as per Geneos Action/Effect conventions and constructs an HTML Email of the dataview the data item is from.

Settings for the Gateway REST connection and defaults for the EMail gateway can be located in dv2email.yaml - either in the working directory or in the user's `.config/dv2email` directory

### Options

```text
      --dir directory      destination directory, defaults to current
  -N, --rowname name       set row name
  -H, --headlines string   order and filter headlines, comma-separated
  -R, --rows string        filter rows, comma-separated
  -O, --order string       order rows, comma-separated column names with optional '+'/'-' suffixes
  -C, --columns string     order and filter columns, comma-separated
```

### Options inherited from parent commands

```text
  -f, --config string     config file (default is $HOME/.config/geneos/dv2email.yaml)
  -D, --dataview string   dataview name, ignored if _VARIBLEPATH set in environment
  -E, --entity string     entity name, ignored if _VARIBLEPATH set in environment
  -S, --sampler string    sampler name, ignored if _VARIBLEPATH set in environment
  -T, --type string       type name, ignored if _VARIBLEPATH set in environment
                          To explicitly select empty/no type use --type/-T ""
```

## SEE ALSO

* [dv2email](dv2email.md)	 - Email a Dataview following Geneos Action/Effect conventions
