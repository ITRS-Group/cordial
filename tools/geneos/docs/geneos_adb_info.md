# `geneos adb info`

Show information about the given ADB file(s)

Without options the output lists the number of each element in the ADB file(s). The counts for each value are also shown, and for attributes the counts for each attribute value are displayed.

The `--show-xpaths`/`-X` option displays a list of all XPaths found in the ADB file(s) instead of the counts for each element.
## Usage

```text
geneos adb info [PATH...] [flags]
```

### Options

```text
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
  -X, --show-xpaths     Show a list of XPaths rather than totals for element values
```

## SEE ALSO

* [geneos adb](geneos_adb.md)	 - ADB File Operations
