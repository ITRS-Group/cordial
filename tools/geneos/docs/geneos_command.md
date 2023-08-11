# `geneos command`

Show Instance Start-up Details

```text
geneos command [TYPE] [NAME...] [flags]
```

Show the command line and environment variables for matching instances.

If given the `--json`/`-j` flag then the output is an array of objects with the details of each instance and the command environment used.

The options `--extras`/`-x` and `--env`/`-e` can be used to add one-off extra command line parameters and environment variables to the command constructors. This is primarily to match the `start` and `restart` commands and for diagnostics and debugging

### Options

```text
  -e, --env NAME=VALUE   Extra environment variable (Repeat as required)
  -x, --extras string    Extra args passed to process, split on spaces and quoting ignored
  -j, --json             JSON formatted output
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
