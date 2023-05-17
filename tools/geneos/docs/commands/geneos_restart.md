## geneos restart

Restart instances

### Synopsis


Restart the matching instances. This is identical to running `geneos
stop` followed by `geneos start` except if the `-a` flag is given
then all matching instances are started regardless of whether they
were stopped by the command. The command also accepts the same flags
as both start and stop.


```
geneos restart [flags] [TYPE] [NAME...]
```

### Options

```
  -a, --all     Start all matching instances, not just those already running
  -F, --force   Force restart of protected instances
  -K, --kill    Force stop by sending an immediate SIGKILL
  -l, --log     Run 'logs -f' after starting instance(s)
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
