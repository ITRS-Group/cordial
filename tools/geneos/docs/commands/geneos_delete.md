## geneos delete

Delete instances

### Synopsis


Delete matching instances.

Instances that are marked `protected` are not deleted without the
`--force`/`-F` option, or they can be unprotected using `geneos
protect -U` first.

Instances that are running are not removed unless the `--stop`/`-S`
option is given.

The instance directory is removed without being backed-up. The user
running the command must have the appropriate permissions and a
partial deletion cannot be protected against.


```
geneos delete [flags] [TYPE] [NAME...]
```

### Options

```
  -S, --stop    Stop instances first
  -F, --force   Force delete of protected instances
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

