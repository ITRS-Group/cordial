# `geneos enable`

Enable matching instances and, if the `--start`/`-S` options is set then start the instance. The `--start`/`-S` flag also starts matching instances that were already enabled, but only if they are not already running.

If called with no arguments `delete` will take no action. If you do want to match all instances then you must use the explicit instance wildcard `all`.

## Usage

```text
geneos enable [flags] [TYPE] [NAME...]
```

### Options

```text
  -S, --start[=true|false]        Start enabled instances
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/geneos/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string           profiles configuration (defaults are ~/.config/geneos/profiles.yaml, /etc/geneos/profiles.yaml)
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
