Create a new key file. With no other options this is written to STDOUT.

To write to a specific file use the `--keyfile`/`-k` option. To write to your user's default key file location use the `--user`/`-u` flag. These options are mutually exclusive.

If the `--shared`/`-S` flag is set then the newly created key file is saved to the shared "keyfiles" directory of component `TYPE` using the base-name of its 8-hexadecimal digit checksum to distinguish it from other key files. In all examples the CRC is shown as `DEADBEEF` in honour of many generations of previous UNIX documentation. There is a very small chance of a checksum clash. If TYPE is not given then all components that support key files are used. When saving key files to shared component directories the contents of the key file are not written to STDOUT, but if combined with `--keyfile/-k` or `--user/-U` then the same keyfile is written to both places.

To update instances to use the new shared keyfile, use the `--update` option. This option is ignored unless the `--shared/-S` option is also used.

An existing key file with the same name will be backed-up using the suffix given with the `--backup`/`-b` option which defaults to `.old`. This is only likely to apply to key files being saved to explicit paths with the `--keyfile` or `--user` options.
