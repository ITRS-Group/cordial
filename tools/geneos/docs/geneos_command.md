# `geneos command`

Show the command line and environment variables for matching instances.

If given the `--json`/`-j` flag then the output is an array of objects with the details of each instance and the command environment used.

The options `--extras`/`-x` and `--env`/`-e` can be used to add one-off extra command line parameters and environment variables to the command constructors. This is primarily to match the `start` and `restart` commands and for diagnostics and debugging

## Usage

```text
geneos command [TYPE] [NAME...]
```

### Options

```text
      --allow-root       allow running as root (not recommended)
  -G, --config string    config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -e, --env NAME=VALUE   Extra environment variable (Repeat as required)
  -x, --extras string    Extra args passed to process, split on spaces and quoting ignored
  -H, --host HOSTNAME    Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
  -j, --json             JSON formatted output
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
