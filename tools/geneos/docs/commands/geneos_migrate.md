# `geneos migrate`

Migrate legacy .rc configuration to new formats

```text
geneos migrate [TYPE] [NAME...] [flags]
```

## Details

Migrate any legacy .rc configuration files to JSON format and
rename the .rc file to .rc.orig. The entries in the new configuration
take on the new labels and are not a direct conversion.

### Options

```text
  -X, --executables   Migrate executables by symlinking to this binary
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
