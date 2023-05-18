# `geneos enable`

Enable instance

```text
geneos enable [flags] [TYPE] [NAME...]
```

## Details

Enable matching instances and, if the `--start`/`-S` options is set
then start the instance. Only those instances that were disabled are
started when the `--start`/`-S` flag is used.

### Options

```text
  -S, --start   Start enabled instances
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
