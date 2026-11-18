# `geneos protect`

The `protect` command marks matching instances as protected. Various operations that affect the state or availability of an instance will be prevented if it is marked `protected`.

To reverse this you must use the same command with the `-U` flag. There is no `unprotect` command. This is by design.

## Usage

```text
geneos protect [flags] [TYPE] [NAME...]
```

### Options

```text
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/geneos/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string           profiles configuration (defaults are ~/.config/geneos/profiles.yaml, /etc/geneos/profiles.yaml)
  -U, --unprotect[=true|false]    unprotect instances
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
