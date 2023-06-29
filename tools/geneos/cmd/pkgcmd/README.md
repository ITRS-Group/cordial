# `geneos package` Subsystem Commands

The `geneos package` commands help you manage ITRS Geneos software releases.

To download and unarchive releases use `geneos package install`. You can also install releases from files you have already downloaded.

Use `geneos package list` to show installed versions and which are linked to which base names (symlinks like `active_prod`).

The `geneos package update` command allows you to select which installed releases to link to which base names and to create new base names.

Finally use `geneos package uninstall` to remove installed packages.

For those commands that use them, `VERSION` options are based on the ITRS Geneos release versioning which is now semantic versioning but older versions will have `GA` and `RA` prefixes:

`[GA]X.Y.Z`

Where X, Y, Z are each ordered in ascending numerical order. If a directory starts `GA` it will be selected over a directory with the same numerical versions. All other directories name formats will result in unexpected behaviour. If multiple installed versions match then the lexically latest match will be used. The chosen match may be much higher than that given on the command line as only installed packages are used in the search.
