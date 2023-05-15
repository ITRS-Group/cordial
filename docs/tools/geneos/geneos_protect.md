## geneos protect

Mark instances as protected

### Synopsis


Mark matcing instances as protected.

To reverse this you must use the same command with the `-U` flag.
There is no `unprotect` command. This is intentional.

Note that you can also manually add or remove the `protected` setting
in an instance configuration file.


```
geneos protect [TYPE] [NAME...] [flags]
```

### Options

```
  -U, --unprotect   unprotect instances
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

