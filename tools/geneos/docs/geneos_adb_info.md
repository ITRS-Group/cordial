# `geneos adb info`

Show information about the given ADB file(s)

Without options the output lists the number of each element in the ADB file(s). The counts for each value are also shown, and for attributes the counts for each attribute value are displayed.

By default, the command reports counts for each element and value.

`--show-xpaths`/`-X` lists unique XPaths only. `--show-urls` lists unique URL targets only. `--show-all` lists both, in separate `XPaths` and `URLs` blocks. Each list is sorted, and URL values are shown without their link wrapper or source type.

Dataitem targets from modifier records and `GeneosURL` records appear in the XPath list. `wwwpage(...)` and `geneos://...` destinations are listed as URL values without wrappers. Web destinations embedded in modifier records are included too.
## Usage

```text
geneos adb info [PATH...] [flags]
```

### Options

```text
      --allow-root[=true|false]    allow running as root (not recommended)
  -G, --config string              config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME              Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string            profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
      --show-all[=true|false]      List both XPaths and URL targets
      --show-urls[=true|false]     List URL targets only
  -X, --show-xpaths[=true|false]   List XPaths only
```

## SEE ALSO

* [geneos adb](geneos_adb.md)	 - ADB File Operations
