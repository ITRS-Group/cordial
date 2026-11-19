# `geneos unset`

The `unset` command removes configuration parameters from matching instances. This command is `unset` rather than `remove` as that is reserved as an alias for the `delete` command.

Unlike the `geneos set` command, where parameters are in the form of KEY=VALUE, there is no way to distinguish a KEY to remove and a possible instance name, so you must use one or more `--key`/`-k` options to unset each simple parameter.

WARNING: Be careful removing keys that are necessary for instances to be manageable. Some keys, if removed, will require manual intervention to remove or fix the old configuration and recreate the instance.

You can also unset values for structured parameters. For `--include`/`-i` options the parameter key is the `PRIORITY` of the include file set while for the other options it is the `NAME`. Note that for structured parameters the `NAME` is case-sensitive. See the usage flags for more details.

Use `--rebuild`/`-r` (also named `--reload`) to rebuild and reload after saving the removals, including when no matching values were removed or the generated configuration is unchanged. Normal `config.rebuild` rules still apply. Use `--force`/`-F` to force the rebuild and reload, including for `config.rebuild=initial`; `--force` implies `--rebuild`. Neither option rebuilds or reloads instances with `config.rebuild=never`. For profiles, saved options are updated and reapplied before the member instances are reloaded.

Without these options, a request that removes nothing remains an error. The existing automatic rebuild and reload behavior is unchanged: supported instances rebuild according to `config.rebuild` and reload when the rebuild reports a change.

## Usage

```text
geneos unset [flags] [TYPE] [NAME...]
```

### Options

```text
  -k, --key KEY                   Unset configuration parameter KEY
                                  (Repeat as required)
  -e, --env NAME                  Remove an environment variable NAME
                                  (Repeat as required)
  -i, --include PRIORITY          Remove an include file with PRIORITY
                                  (Repeat as required, gateways only)
  -g, --gateway NAME              Remove the gateway NAME
                                  (Repeat as required, san and floating only)
  -a, --attribute NAME            Remove the attribute NAME
                                  (Repeat as required, san only)
  -t, --type NAME                 Remove the type NAME
                                  (Repeat as required, san only)
  -v, --variable NAME             Remove the variable NAME
                                  (Repeat as required, san only)
  -M, --entity NAME               Remove a managed entity NAME
                                  (Repeat as required, san only)
  -r, --rebuild[=true|false]      Rebuild and reload after unsetting values, even if unchanged
      --reload[=true|false]       Alias for --rebuild
  -F, --force[=true|false]        Force rebuild and reload, except when config.rebuild=never
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/geneos/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string           profiles configuration (defaults are ~/.config/geneos/profiles.yaml, /etc/geneos/profiles.yaml)
```

## Examples

```bash
geneos unset gateway GW1 -k aesfile
geneos unset san -g Gateway1

```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
