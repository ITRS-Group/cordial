# `geneos config show`

The show command outputs the current configuration for the `geneos` program in JSON format. By default it shows the processed values from the on-disk copy of your program configuration and not the final configuration that the running program uses, which includes many built-in defaults.

Using the `--all`/`-a` options then the output is the full running configuration from loading and merging all configuration files that apply, including internal and external defaults.

If any arguments are given then they are treated as a list of keys to limit the output to just those keys that match and have a non-nil value.

No values that are encrypted are shown decrypted.

## Usage

```text
geneos config show [KEY...] [flags]
```

### Options

```text
  -a, --all             Show all the parameters including all defaults
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos config](geneos_config.md)	 - Configure Command Behaviour
