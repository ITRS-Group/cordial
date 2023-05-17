## geneos config set

Set program configuration

### Synopsis


Set configuration parameters for the `geneos` program.

Each value is in the form of KEY=VALUE where key is the configuration
item and value an arbitrary string value. Where a KEY is in a
hierarchy use a dot (`.`) as the delimiter.

While you can set arbitrary keys only some have any meaning. The most
important one is `geneos`, the path to the root directory of the
Geneos installation managed by the program. If you change or remove
this value you may break the functionality of the program, so please
be careful.

For an explanation of the various configuration parameters see the
main documentation.


```
geneos config set [KEY=VALUE...] [flags]
```

### Examples

```

geneos config set geneos="/opt/geneos"
geneos config set config.rebuild=always

```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

### SEE ALSO

* [geneos config](geneos_config.md)	 - Configure geneos command environment

