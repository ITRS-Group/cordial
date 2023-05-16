## geneos clean

Clean-up instance directories

### Synopsis


Clean-up instance directories by removing old log & config file backups
from the working directory of the targetted instances, without affecting
the running instances.

If run with the `-F` (or `--full`) option, `geneos clean` will stop the 
targetted instances, remove all non-essential files from the working 
directory of the targetted instances and restart the targetted instances.

**Note**: Files removed by `geneos clean` are defined in the geneos main 
configuration file `geneos.json` as `[TYPE]CleanList`.
Files removed by `geneos clean -F` or `geneos clean --full` are defined
in the geneos main configuration file `geneos.json` as `[TYPE]PurgeList`.
Both these lists are formatted as a PathListSeparator (typically a colon) 
separated list of file globs.


```
geneos clean [flags] [TYPE] [NAME...]
```

### Examples

```

# Delete old logs and config file backups without affecting the running
# instance
geneos clean gateway Gateway1
# Stop all netprobes and remove all non-essential files from working 
# directories, then restart netprobes
geneos clean --full netprobe

```

### Options

```
  -F, --full   Perform a full clean. Removes more files than basic clean and restarts instances
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

