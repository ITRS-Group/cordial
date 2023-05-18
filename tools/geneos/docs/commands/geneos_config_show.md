# geneos config show

Show program configuration

```text
geneos config show [KEY...] [flags]
```

## Details

The show command outputs the current configuration for the `geneos`
program in JSON format. It shows the processed values from the
on-disk copy of your program configuration and not the final
configuration that the running program uses, which includes many
built-in defaults.

If any arguments are given then they are treated as a list of keys to
limit the output to just those keys that match and have a non-nil value.

### Options

```text
  -a, --all   Show all the parameters including all defaults
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos config](geneos_config.md)	 - Configure geneos command environment
