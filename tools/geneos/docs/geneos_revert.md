# `geneos revert`

The `revert` command will revert the `.rc.orig` suffixed configuration file for all matching instances.

For any instance that is `protected` this will fail and an error reported.

The original file is never updated and any changes made since the original migration will be lost. The new configuration file will be deleted.

If there is already a configuration file with a `.rc` suffix then the command will remove any `.rc.orig` and new configuration files while leaving the existing file unchanged.

If called with the `--executables`/`-X` option then instead of instance configurations the command will remove any symbolic links from legacy `ctl` command in `${GENEOS_HOME}/bin` that point to the command.

## Usage

```text
geneos revert [--executables|-X] | [TYPE] [NAME...]
```

### Options

```text
  -X, --executables     Revert 'ctl' executables
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
