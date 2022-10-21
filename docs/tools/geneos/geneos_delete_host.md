## geneos delete host

Delete a remote host configuration

### Synopsis


Delete the local configuration referring to a remote host.


```
geneos delete host [flags] NAME...
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

* [geneos delete](geneos_delete.md)	 - Delete an instance. Instance must be stopped

