## geneos unset

Unset a configuration value

### Synopsis

Unset a configuration value.
	
This command has been added to remove the confusing negation syntax in set

```
geneos unset [FLAGS] [TYPE] [NAME...]
```

### Examples

```

geneos unset gateway GW1 -k aesfile
geneos unset san -g Gateway1

```

### Options

```
  -k, --key SETTING         Unset a configuration key item
  -e, --env SETTING         Remove an environment variable of NAME
  -i, --include SETTING     Remove an include file in the format PRIORITY
  -g, --gateway SETTING     Remove gateway NAME
  -a, --attribute SETTING   Remove an attribute of NAME
  -t, --type SETTING        Remove the type NAME
  -v, --variable SETTING    Remove a variable of NAME
  -h, --help                help for unset
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
* [geneos unset global](geneos_unset_global.md)	 - 
* [geneos unset user](geneos_unset_user.md)	 - 

