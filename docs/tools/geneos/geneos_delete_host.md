## geneos delete host

A brief description of your command

### Synopsis

A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.

```
geneos delete host [-F] [TYPE] NAME...
```

### Options

```
  -F, --force   Delete instances without checking if disabled
  -R, --all     Recursively delete all instances on the host before removing the host config
  -S, --stop    Stop all instances on the host before deleting the local entry
  -h, --help    help for host
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos delete](geneos_delete.md)	 - Delete an instance. Instance must be stopped

