## geneos revert

Revert migration of .rc files from backups

### Synopsis


Revert migration of legacy .rc files to JSON if the .rc.orig backup
file still exists. Any changes to the instance configuration since
initial migration will be lost as the contents of the .rc file is
never changed.


```
geneos revert [TYPE] [NAME...] [flags]
```

### Options

```
  -X, --executables   Revert 'ctl' executables
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

