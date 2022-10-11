## geneos delete

Delete an instance. Instance must be stopped

### Synopsis

Delete the matching instances. This will only work on
instances that are disabled to prevent accidental deletion. The
instance directory is removed without being backed-up. The user
running the command must have the appropriate permissions and a
partial deletion cannot be protected against.

```
geneos delete [-F] [TYPE] [NAME...]
```

### Options

```
  -F, --force   Force delete of instances
  -h, --help    help for delete
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
* [geneos delete host](geneos_delete_host.md)	 - A brief description of your command

