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
      --columns string     filter columns, comma-separated string
      --dir directory      destination directory, defaults to current
      --headlines string   filter headlines, comma-separated string
      --rowname name       set row name
      --rows string        filter rows, comma-separated string
```

### Options inherited from parent commands

```text
  -f, --config string     config file (default is $HOME/.config/geneos/dv2email.yaml)
  -D, --dataview string   dataview name, ignored if _VARIBLEPATH set in environment
  -E, --entity string     entity name, ignored if _VARIBLEPATH set in environment
  -S, --sampler string    sampler name, ignored if _VARIBLEPATH set in environment
  -T, --type string       type name, ignored if _VARIBLEPATH set in environment
```

## SEE ALSO

* [dv2email](dv2email.md)	 - Email a Dataview following Geneos Action/Effect conventions
