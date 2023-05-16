## geneos host show

Show details of remote host configuration

### Synopsis


Show details of remote host configurations. If no names are supplied
then all configured hosts are shown.

The output is always unprocessed, and so any values in `expandable`
format are left as-is. This protects, for example, SSH passwords from
being accidentally shown in clear text.


```
geneos host show [flags] [NAME...]
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos host](geneos_host.md)	 - Manage remote host settings

