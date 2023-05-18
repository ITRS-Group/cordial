# `geneos protect`

Mark instances as protected

```text
geneos protect [TYPE] [NAME...] [flags]
```

## Details

Mark matching instances as protected. Various operations that affect
the state or availability of an instance will be prevented if it is
marked `protected`.

To reverse this you must use the same command with the `-U` flag.
There is no `unprotect` command. This is by design.

### Options

```text
  -U, --unprotect   unprotect instances
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
