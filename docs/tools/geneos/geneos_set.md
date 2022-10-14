## geneos set

Set instance configuration parameters

### Synopsis


Set configuration item values in global, user, or for a specific
instance.

To set "special" items, such as Environment variables or Attributes you should
now use the specific flags and not the old special syntax.

The "set" command does not rebuild any configuration files for instances.
Use "rebuild" to do this.


```
geneos set [FLAGS] [TYPE] [NAME...] KEY=VALUE [KEY=VALUE...] [flags]
```

### Options

```
  -e, --env NAME                      (all components) Add an environment variable in the format NAME=VALUE
  -i, --include PRIORITY:{URL|PATH}   (gateways) Add an include file in the format PRIORITY:PATH
  -g, --gateway HOSTNAME:PORT         (sans) Add a gateway in the format NAME:PORT
  -a, --attribute NAME                (sans) Add an attribute in the format NAME=VALUE
  -t, --type NAME                     (sans) Add a type NAME
  -v, --variable [TYPE:]NAME=VALUE    (sans) Add a variable in the format [TYPE:]NAME=VALUE
  -h, --help                          help for set
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
* [geneos set global](geneos_set_global.md)	 - Set global configuration parameters
* [geneos set user](geneos_set_user.md)	 - Set user configuration parameters

