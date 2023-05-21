# `geneos protect`

Mark instances as protected

```text
geneos protect [TYPE] [NAME...] [flags]
```

Mark matching instances as protected. Various operations that affect the
state or availability of an instance will be prevented if it is marked
`protected`.

To reverse this you must use the same command with the `-U` flag. There
is no `unprotect` command. This is by design.

### Options

```text
  -U, --unprotect   unprotect instances
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
