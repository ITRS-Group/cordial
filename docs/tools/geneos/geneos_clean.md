## geneos clean

Clean-up instance directories

### Synopsis


Clean-up instance directories, also restarting instances if doing a
full clean using `-F`. The patterns of files and directories that are
cleaned up are set in the global configuration as `[TYPE]CleanList`
and `[TYPE]PurgeList` and can be seen using the `geneos show`
command, and changed using `geneos set`. The format is a
PathListSeperator (typically a colon) separated list of file globs.


```
geneos clean [flags] [TYPE] [NAME...]
```

### Examples

```

# delete old logs and config file backups without affecting running instance
geneos clean gateway Gateway1
# stop all netprobes and remove all non-essential files from working directories,
# then restart
geneos clean --full netprobe

```

### Options

```
  -F, --full   Perform a full clean. Removes more files than basic clean and restarts instances
  -h, --help   help for clean
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

