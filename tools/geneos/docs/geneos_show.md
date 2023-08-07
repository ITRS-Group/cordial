# `geneos show`

Show Instance Configuration

```text
geneos show [flags] [TYPE] [NAME...]
```

Show the configuration for matching instances or show the Geneos configuration used by an instance (depending on options selected below).

By default the configuration used to manage the matching instances is output as a JSON array of objects. Each instance object contains
`instance` metadata and a `configuration` object.

The output can be written to a file using the `--output`/`-o` option.

For instance types that have Geneos configuration files, i.e. Gateways, Self-Announcing and Floating Netprobes, the `--setup`/`-s` option can be used to view these.

For Gateways the `--merge`/`-m` option tries to output a merged configuration using the Gateway `-dump-xml` option, but this is subject to unexpected behaviour when not all remote include files are reachable and so on.

The `show` command can also be used trigger validation of Gateway configuration(s) with the `--validate`/`--V` option. When validating you can pass a Gateway Hooks directory with `--hooks-dir` which can be used to further trigger external processing of the Gateway configuration. You cannot use `--merge` and `--validate` at the same time as during validation the Gateway will internally merge the configuration.

For normal output each instance's underlying configuration is in an object key `configuration`. Only the objects in this `configuration` key are stored in the instance's actual configuration file and this is the root for all parameter names used by other commands, i.e. for a value under `configuration.licdsecure` the parameter you would use for a `geneos set` command is just `licdsecure`. Confusingly there is a `configuration.config` object, used for template support. Other run-time information is shown under the `instance` key and includes the instance name, the host it is configured on, it's type and so on.

By default the interpolated ("expandable" values are expanded) values are shown. The see the underlying value use the `--raw`/`-r` option.

No values that are encrypted are shown decrypted with or without the `--raw`/`-r` option.
### Options

```text
  -o, --output string   Output file, default stdout
  -r, --raw             Show raw (unexpanded) configuration values
  -s, --setup           Show the instance Geneos configuration file, if any
  -m, --merge           Merge Gateway configurations using the Gateway -dump-xml flag
  -V, --validate        Validate Gateway configurations using the Gateway -validate flag
      --hooks string    Hooks directory
                        (may clash with instance parameters if set for normal execution)
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
