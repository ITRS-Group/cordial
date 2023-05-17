## geneos enable

Enable instances

### Synopsis


Mark any matching instances as enabled and if the `-S` flag is given
then start the instance. Only those instances that were disabled are started
when the `-S` flag is used.


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
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
