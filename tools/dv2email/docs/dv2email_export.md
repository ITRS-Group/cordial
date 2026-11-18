# `dv2email export`

Write Dataview(s) to local files

## Usage

```text
dv2email export [flags]
```

### Options

```text
      --dir directory      destination directory, defaults to current
  -N, --rowname name       set row name
  -H, --headlines string   order and filter headlines, comma-separated
  -R, --rows string        filter rows, comma-separated
  -O, --order string       order rows, comma-separated column names with optional '+'/'-' suffixes
  -C, --columns string     order and filter columns, comma-separated
  -f, --config string      config file (default is $HOME/.config/geneos/dv2email.yaml)
  -D, --dataview string    dataview name, ignored if _VARIABLEPATH set in environment
  -E, --entity string      entity name, ignored if _VARIABLEPATH set in environment
  -S, --sampler string     sampler name, ignored if _VARIABLEPATH set in environment
  -T, --type string        type name, ignored if _VARIABLEPATH set in environment
                           To explicitly select empty/no type use --type/-T ""
```

## SEE ALSO

* [dv2email](dv2email.md)	 - Email a Dataview following Geneos Action/Effect conventions
