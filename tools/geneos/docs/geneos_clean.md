# `geneos clean`

Clean the working directory for matching instances.

The default behaviour is to leave the instance running and only remove files known to be old or inactive.

The `--full`/`-F` option has been deprecated, use the `reset` command instead. It will remove more files than the basic clean and restart the instance.

**Note**: Files removed by `geneos clean` are defined in the geneos main configuration file `geneos.json` as `[TYPE]::clean`. Files removed by `geneos clean -F` or `geneos clean --full` are defined in the geneos main configuration file `geneos.json` as `[TYPE]::purge`. Both these lists are formatted as a PathListSeparator (typically a colon) separated list of file pattern globs (not regular expressions).

## Usage

```text
geneos clean [flags] [TYPE] [NAME...]
```

### Options

```text
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## Examples

```bash
# Delete old logs and config file backups without affecting the running
# instance
geneos clean gateway Gateway1

```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
