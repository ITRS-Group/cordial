## geneos disable

Disable instances

### Synopsis



Mark any matching instances as disabled. The instances are also
stopped.


```
geneos disable [TYPE] [NAME...] [flags]
```

### Options

```
  -S, --stop    Stop instances
  -F, --force   Force disable instances
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

