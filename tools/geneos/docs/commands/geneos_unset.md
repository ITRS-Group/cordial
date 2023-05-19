# `geneos unset`

Unset configuration parameters

```text
geneos unset [flags] [TYPE] [NAME...]
```

Unset (remove) configuration parameters from matching instances. This
command is `unset` rather than `remove` as that is reserved as an alias
for the `delete` command.

Unlike the `geneos set` command, where parameters are in the form of
KEY=VALUE, there is no way to distinguish a KEY to remove and a possible
instance name, so you must use one or more `--key`/`-k` options to unset
each simple parameter.

WARNING: Be careful removing keys that are necessary for instances to be
manageable. Some keys, if removed, will require manual intervention to
remove or fox the old configuration and recreate the instance.

You can also unset values for structured parameters. For
`--include`/`-i` options the parameter key is the `PRIORITY` of the
include file set while for the other options it is the `NAME`. Note that
for structured parameters the `NAME` is case-sensitive.

### Options

```text
  -k, --key KEY            Unset configuration parameter KEY
                           (Repeat as required)
  -e, --env NAME           Remove an environment variable NAME
                           (Repeat as required)
  -i, --include PRIORITY   Remove an include file with PRIORITY
                           (Repeat as required, gateways only)
  -g, --gateway NAME       Remove the gateway NAME
                           (Repeat as required, san and floating only)
  -a, --attribute NAME     Remove the attribute NAME
                           (Repeat as required, san only)
  -t, --type NAME          Remove the type NAME
                           (Repeat as required, san only)
  -v, --variable NAME      Remove the variable NAME
                           (Repeat as required, san only)
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

* [geneos](geneos.md)	 - Take control of your Geneos environments
