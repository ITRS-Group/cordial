# `geneos migrate`

Migrate configurations

```text
geneos migrate [TYPE] [NAME...] [flags]
```

## Details

By default the `migrate` command will convert legacy `.rc` format
files to JSON and named the old file to end `.rc.orig`. The `revert`
command can be used to restore these backup files. If you run a full
clean (`geneos clean -F`) then it is likely these backup files will
be removed.

The `--executables`/`-X` option instead creates symbolic links in the
${GENEOS_HOME}/bin directory for names that match the original `ctl`
scripts pointing back to the `geneos` program.

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

* [geneos](geneos.md)	 - Take control of your Geneos environments
