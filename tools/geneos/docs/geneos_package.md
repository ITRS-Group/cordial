# `geneos package`

The `package` sub-system commands help you manage Geneos software releases.

For those commands that use them, `VERSION` options are based on the ITRS Geneos release versioning which is now semantic versioning but older versions will have `GA` and `RA` prefixes:

`[GA]X.Y.Z`

Where X, Y, Z are each ordered in ascending numerical order. If a directory starts `GA` it will be selected over a directory with the same numerical versions. All other directories name formats will result in unexpected behaviour. If multiple installed versions match then the lexically latest match will be used. The chosen match may be much higher than that given on the command line as only installed packages are used in the search.


## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos package install`](geneos_package_install.md)	 | Install Geneos releases |
| [`geneos package list / ls`](geneos_package_list.md)	 | List packages available for update command |
| [`geneos package uninstall / delete / remove / rm`](geneos_package_uninstall.md)	 | Uninstall Geneos releases |
| [`geneos package update`](geneos_package_update.md)	 | Update the active version of installed Geneos package |

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
