# `geneos config unset`

Unset removes the program configuration value for any arguments given on the command line.

No validation is done and there if you mistype a key name it is still considered valid to remove an non-existing key.

## Usage

```text
geneos config unset [KEY...] [flags]
```

### Options

```text
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos config](geneos_config.md)	 - Configure Command Behaviour
