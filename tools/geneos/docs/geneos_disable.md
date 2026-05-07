# `geneos disable`

Mark any matching instances as disabled. The instances are also stopped.

If called with no arguments `disable` will take no action. If you do want to match all instances then you must use the explicit instance wildcard `all`.

## Usage

```text
geneos disable [TYPE] [NAME...] [flags]
```

### Options

```text
  -S, --stop            Stop instances
  -F, --force           Force disable instances
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
