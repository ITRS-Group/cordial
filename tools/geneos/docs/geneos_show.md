# `geneos show`

Show Instance Configuration

```text
geneos show [flags] [TYPE] [NAME...]
```

Show the configuration for matching instances.

At the moment this is in JSON format and is output as a single,
concatenated JSON array of object literals, one per instance.

Each instance's underlying configuration is in an object key
`configuration`. Only the objects in this `configuration` key are stored
in the instance's actual configuration file and this is the root for all
parameter names used by other commands, i.e. for a value under
`configuration.licdsecure` the parameter you would use for a `geneos
set` command is just `licdsecure`. Confusingly there is a
`configuration.config` object, used for template support. Other run-time
information is shown under the `instance` key and includes the instance
name, the host it is configured on, it's type and so on.

By default the interpolated ("expandable" values are expanded) values
are shown. The see the underlying value use the `--raw`/`-r` option.

No values that are encrypted are shown decrypted with or without the
`--raw`/`-r` option.
### Options

```text
  -r, --raw   Show raw (unexpanded) configuration values
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
