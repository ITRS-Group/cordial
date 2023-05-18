# `geneos disable`

Disable instances

```text
geneos disable [TYPE] [NAME...] [flags]
```

## Details


Mark any matching instances as disabled. The instances are also
stopped.

### Options

```text
  -S, --stop    Stop instances
  -F, --force   Force disable instances
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
