The `package` sub-system commands help you manage Geneos software releases.

For those commands that use them, `VERSION` options are based on the ITRS Geneos release versioning which is now semantic versioning but older versions will have `GA` and `RA` prefixes:

`[GA]X.Y.Z`

Where X, Y, Z are each ordered in ascending numerical order. If a directory starts `GA` it will be selected over a directory with the same numerical versions. All other directories name formats will result in unexpected behaviour. If multiple installed versions match then the lexically latest match will be used. The chosen match may be much higher than that given on the command line as only installed packages are used in the search.
