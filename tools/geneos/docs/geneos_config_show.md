# `geneos config show`

The show command outputs the current configuration for the `geneos` program in JSON format. By default it shows the processed values from the on-disk copy of your program configuration and not the final configuration that the running program uses, which includes many built-in defaults.

Using the `--all`/`-a` options then the output is the full running configuration from loading and merging all configuration files that apply, including internal and external defaults.

If any arguments are given then they are treated as a list of keys to limit the output to just those keys that match and have a non-nil value.

No values that are encrypted are shown decrypted.

```text
geneos config show [KEY...] [flags]
```

### Options

```text
  -a, --all   Show all the parameters including all defaults
```

## SEE ALSO

* [geneos config](geneos_config.md)	 - Configure Command Behaviour
