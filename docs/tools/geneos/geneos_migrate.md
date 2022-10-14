## geneos migrate

Migrate legacy .rc configuration to new formats

### Synopsis


Migrate any legacy .rc configuration files to JSON format and
rename the .rc file to .rc.orig. The entries in the new configuration
take on the new labels and are not a direct conversion.


```
geneos migrate [TYPE] [NAME...] [flags]
```

### Options

```
  -h, --help   help for migrate
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

