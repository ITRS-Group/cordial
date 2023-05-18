# geneos unset

Unset a configuration value

```text
geneos unset [flags] [TYPE] [NAME...]
```

## Details

Unset a configuration value.
	
This command has been added to remove the confusing negation syntax
in the `set` command

### Options

```text
  -k, --key SETTING        Unset a configuration key item
  -e, --env NAME           Remove an environment variable NAME
  -i, --include PRIORITY   (gateways) Remove an include file withPRIORITY
  -g, --gateway NAME       (san) Remove the gateway NAME
  -a, --attribute NAME     (san) Remove the attribute NAME
  -t, --type NAME          (san) Remove the type NAME
  -v, --variable NAME      (san) Remove the variable NAME
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## Examples

```bash
geneos unset gateway GW1 -k aesfile
geneos unset san -g Gateway1
```

## SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
