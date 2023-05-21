# `geneos clean`

Clean-up Instance Directories

```text
geneos clean [flags] [TYPE] [NAME...]
```

Clean the working directory for matching instances.

The default behaviour is to leave the instance running and only remove
known to be inactive files.

With the `--full`/`-F` option, the command will stop the instance,
remove all non-essential files from the working directory of the
instance and restart the instance.

**Note**: Files removed by `geneos clean` are defined in the geneos main
configuration file `geneos.json` as `[TYPE]CleanList`. Files removed by
`geneos clean -F` or `geneos clean --full` are defined in the geneos
main configuration file `geneos.json` as `[TYPE]PurgeList`. Both these
lists are formatted as a PathListSeparator (typically a colon) separated
list of file globs.

### Options

```text
  -F, --full   Perform a full clean. Removes more files than basic clean and restarts instances
```

## Examples

```bash
# Delete old logs and config file backups without affecting the running
# instance
geneos clean gateway Gateway1
# Stop all netprobes and remove all non-essential files from working 
# directories, then restart netprobes
geneos clean --full netprobe

```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
