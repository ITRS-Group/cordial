# `geneos enable`

Enable matching instances and, if the `--start`/`-S` options is set then start the instance. Only those instances that were disabled are started when the `--start`/`-S` flag is used.

If called with no arguments `delete` will take no action. If you do want to match all instances then you must use the explicit instance wildcard `all`.

## Usage

```text
geneos enable [flags] [TYPE] [NAME...]
```

### Options

```text
  -S, --start           Start enabled instances
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
