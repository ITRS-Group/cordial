# `geneos delete`

Delete matching instances.

Instances that are marked `protected` are not deleted without the `--force`/`-F` option, or they can be unprotected using `geneos protect -U` first.

Instances that are running are not removed unless the `--stop`/`-S` option is given.

The instance directory is removed without being backed-up. The user running the command must have the appropriate permissions and a partial deletion cannot be protected against.

If called with no arguments `delete` will take no action. If you do want to match all instances then you must use the explicit instance wildcard `all`.

## Usage

```text
geneos delete [flags] [TYPE] [NAME...]
```

### Options

```text
  -S, --stop            Stop instances first
  -F, --force           Force delete of protected instances
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
