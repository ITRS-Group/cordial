# `geneos delete`

Delete instances

```text
geneos delete [flags] [TYPE] [NAME...]
```

Delete matching instances.

Instances that are marked `protected` are not deleted without the
`--force`/`-F` option, or they can be unprotected using `geneos protect
-U` first.

Instances that are running are not removed unless the `--stop`/`-S`
option is given.

The instance directory is removed without being backed-up. The user
running the command must have the appropriate permissions and a partial
deletion cannot be protected against.

### Options

```text
  -S, --stop    Stop instances first
  -F, --force   Force delete of protected instances
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
