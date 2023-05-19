Clean the working directories for all matching instances.

The default behaviour is to leave the instance running and only inactive files
are removed.

With the `--full`/`-F` option, the command will stop the
instance, remove all non-essential files from the working
directory of the instance and restart the instance.

**Note**: Files removed by `geneos clean` are defined in the geneos main
configuration file `geneos.json` as `[TYPE]CleanList`. Files removed by
`geneos clean -F` or `geneos clean --full` are defined in the geneos main
configuration file `geneos.json` as `[TYPE]PurgeList`. Both these lists are
formatted as a PathListSeparator (typically a colon) separated list of file
globs.
