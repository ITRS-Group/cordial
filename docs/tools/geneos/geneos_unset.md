## geneos unset

Unset a configuration value

### Synopsis


Unset a configuration value.
	
This command has been added to remove the confusing negation syntax
in the `set` command


```
geneos unset [flags] [TYPE] [NAME...]
```

### Examples

```

geneos unset gateway GW1 -k aesfile
geneos unset san -g Gateway1

```

### Options

```
  -k, --key SETTING        Unset a configuration key item
  -e, --env NAME           Remove an environment variable NAME
  -i, --include PRIORITY   (gateways) Remove an include file withPRIORITY
  -g, --gateway NAME       (san) Remove the gateway NAME
  -a, --attribute NAME     (san) Remove the attribute NAME
  -t, --type NAME          (san) Remove the type NAME
  -v, --variable NAME      (san) Remove the variable NAME
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
* [geneos unset global](geneos_unset_global.md)	 - Unset a global parameter
* [geneos unset user](geneos_unset_user.md)	 - Unset a user parameter

