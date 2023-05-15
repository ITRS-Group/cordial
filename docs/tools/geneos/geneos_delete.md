## geneos delete

Delete an instance. Instance must be stopped

### Synopsis


Delete the matching instances. This will only work on instances that
are disabled, or if the `-F` flag is given, to prevent accidental
deletion. The instance directory is removed without being backed-up.
The user running the command must have the appropriate permissions
and a partial deletion cannot be protected against.


```
geneos delete [flags] [TYPE] [NAME...]
```

### Options

```
  -F, --force   Force delete of instances
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
* [geneos delete host](geneos_delete_host.md)	 - Alias for `host delete`

