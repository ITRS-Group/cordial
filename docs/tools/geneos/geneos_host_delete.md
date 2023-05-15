## geneos host delete

Delete a remote host configuration

### Synopsis


Delete the local configuration referring to a remote host.


```
geneos host delete [flags] NAME...
```

### Options

```
  -F, --force   Delete instances without checking if disabled
  -R, --all     Recursively delete all instances on the host before removing the host config
  -S, --stop    Stop all instances on the host before deleting the local entry
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos host](geneos_host.md)	 - Manage remote host settings

