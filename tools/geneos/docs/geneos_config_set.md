# `geneos config set`

Set configuration parameters for the `geneos` program.

Each value is in the form of `KEY=VALUE` where key is the configuration item and value an arbitrary string value. Where a `KEY` is in a hierarchy use a dot (`.`) as the delimiter.

While you can set arbitrary keys only some have any meaning. The most important one is `geneos`, the path to the root directory of the Geneos installation managed by the program. If you change or remove this value you may break the functionality of the program, so please be careful.

For an explanation of the various configuration parameters see the main documentation.

## Usage

```text
geneos config set [KEY=VALUE...] [flags]
```

### Options

```text
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## Examples

```bash
geneos config set geneos="/opt/geneos"

```

## SEE ALSO

* [geneos config](geneos_config.md)	 - Configure Command Behaviour
