# `geneos disable`

Mark any matching instances as disabled. The instances are also stopped.

If called with no arguments `disable` will take no action. If you do want to match all instances then you must use the explicit instance wildcard `all`.

## Usage

```text
geneos disable [TYPE] [NAME...] [flags]
```

### Options

```text
  -S, --stop[=true|false]         Stop instances
  -F, --force[=true|false]        Force disable instances
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string           profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
