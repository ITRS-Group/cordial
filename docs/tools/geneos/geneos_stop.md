## geneos stop

Stop instances

### Synopsis


Stop one or more matching instances. Unless the -K
flag is given, a SIGTERM is sent and if the instance is
still running after a few seconds then a SIGKILL is sent. If the
`-K` flag is given the instance(s) are immediately terminated with
a `SIGKILL`.


```
geneos stop [flags] [TYPE] [NAME...]
```

### Options

```
  -K, --kill   Force immediate stop by sending an immediate SIGKILL
  -h, --help   help for stop
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

