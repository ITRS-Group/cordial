## geneos enable

Enable instance

### Synopsis


Enable matching instances and, if the `--start`/`-S` options is set
then start the instance. Only those instances that were disabled are
started when the `--start`/`-S` flag is used.


```
geneos enable [flags] [TYPE] [NAME...]
```

### Options

```
  -S, --start   Start enabled instances
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

