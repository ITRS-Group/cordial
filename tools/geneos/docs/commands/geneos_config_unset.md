# geneos config unset

Unset a program parameter

```text
geneos config unset [KEY...] [flags]
```

## Details

Unset removes the program configuration value for any arguments given
on the command line.

No validation is done and there if you mistype a key name it is still
considered valid to remove an non-existing key.

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos config](geneos_config.md)	 - Configure geneos command environment
