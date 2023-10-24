# `geneos disable`

Mark any matching instances as disabled. The instances are also stopped.

If called with no arguments `disable` will take no action. If you do want to match all instances then you must use the explicit instance wildcard `all`.

```text
geneos disable [TYPE] [NAME...] [flags]
```

### Options

```text
  -S, --stop    Stop instances
  -F, --force   Force disable instances
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
