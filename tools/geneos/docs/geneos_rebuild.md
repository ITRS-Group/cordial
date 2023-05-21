# `geneos rebuild`

Rebuild instance configuration files

```text
geneos rebuild [flags] [TYPE] [NAME...]
```

All matching instances whose TYPE supported templates for configuration
file will have them rebuilt depending on the `config.rebuild` setting
for each instance.

The values for the `config.rebuild` option are: `never`, `initial` and
`always`. The default value depends on the TYPE; For Gateways it is
`initial` and for SANs and Floating Netprobes it is `always`.

You can force a rebuild for an instance that has the `config.rebuild`
set to `initial` by using the `--force`/`-F` option. Instances with a
`never` setting are never rebuilt.

The change this use something like `geneos set gateway
config.rebuild=always`

Instances will not normally update their settings when the configuration
file changes, although there are options for both Gateways and Netprobes
to do this, so you can trigger a configuration reload with the
`--reload`/`-r` option. This will send the appropriate signal to
matching instances regardless of the underlying configuration being
updated or not.

The templates use for each TYPE are stored in a `templates/` directory
under each TYPE. If you do not have templates because you are adopting
an existing installation or you have upgraded the `geneos` program and
want updated templates then run `geneos init template` to overwrite
existing file with the built-in ones.

### Options

```text
  -F, --force    Force rebuild
  -r, --reload   Reload instances after rebuild
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
