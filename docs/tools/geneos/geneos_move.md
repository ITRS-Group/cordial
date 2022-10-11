## geneos move

Move (or rename) instances

### Synopsis

Move (or rename) instances. As any existing legacy .rc
file is never changed, this will migrate the instance from .rc to
JSON. The instance is stopped and restarted after the instance is
moved. It is an error to try to move an instance to one that already
exists with the same name.

If the component support Rebuild then this is run after the move but
before the restart. This allows SANs to be updated as expected.

Moving across hosts is supported.

```
geneos move [TYPE] SOURCE DESTINATION
```

### Options

```
  -h, --help   help for move
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

