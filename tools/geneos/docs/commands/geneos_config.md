## geneos config

Configure geneos command environment

### Synopsis


The commands in the `config` subsystem allow you to control the
environment of the `geneos` program itself. Please see the
descriptions of the commands below for more information.

If you run this command directly then you will either be shown the
output of `geneos config show` or `geneos config set [ARGS]` if you
supply any further arguments that contain an "=".


```
geneos config [flags]
```

### Examples

```

geneos config
geneos config geneos=/opt/itrs

```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
* [geneos config set](geneos_config_set.md)	 - Set program configuration
* [geneos config show](geneos_config_show.md)	 - Show program configuration
* [geneos config unset](geneos_config_unset.md)	 - Unset a program parameter

