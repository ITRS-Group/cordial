# `geneos enable`

Enable instance

```text
geneos enable [flags] [TYPE] [NAME...]
```

Enable matching instances and, if the `--start`/`-S` options is set then start the instance. Only those instances that were disabled are started when the `--start`/`-S` flag is used.

If called with no arguments `delete` will take no action. If you do want to match all instances then you must use the explicit instance wildcard `all`.

### Options

```text
  -S, --start   Start enabled instances
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
