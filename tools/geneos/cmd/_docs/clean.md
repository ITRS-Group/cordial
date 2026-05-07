Clean the working directory for matching instances.

The default behaviour is to leave the instance running and only remove files known to be old or inactive.

The `--full`/`-F` option has been deprecated, use the `reset` command instead. It will remove more files than the basic clean and restart the instance.

**Note**: Files removed by `geneos clean` are defined in the geneos main configuration file `geneos.json` as `[TYPE]::clean`. Files removed by `geneos clean -F` or `geneos clean --full` are defined in the geneos main configuration file `geneos.json` as `[TYPE]::purge`. Both these lists are formatted as a PathListSeparator (typically a colon) separated list of file pattern globs (not regular expressions).
