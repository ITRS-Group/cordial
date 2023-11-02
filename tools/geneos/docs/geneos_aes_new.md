# `geneos aes new`

Create a new key file. With no other options this is written to STDOUT.

To write to a specific file use the `--keyfile`/`-k` option. To write to your user's default key file location use the `--user`/`-u` flag. These options are mutually exclusive.

If the `--shared`/`-S` flag is set then the newly created key file is saved to the shared "keyfiles" directory of component `TYPE` using the base-name of its 8-hexadecimal digit checksum to distinguish it from other key files. In all examples the CRC is shown as `DEADBEEF` in honour of many generations of previous UNIX documentation. There is a very small chance of a checksum clash. If TYPE is not given then all components that support key files are used. When saving key files to shared component directories the contents of the key file are not written to STDOUT, but if combined with `--keyfile/-k` or `--user/-U` then the same keyfile is written to both places.

To update instances to use the new shared keyfile, use the `--update` option. This option is ignored unless the `--shared/-S` option is also used.

An existing key file with the same name will be backed-up using the suffix given with the `--backup`/`-b` option which defaults to `.old`. This is only likely to apply to key files being saved to explicit paths with the `--keyfile` or `--user` options.

```text
geneos aes new [flags] [TYPE] [NAME...]
```

### Options

```text
  -k, --keyfile KEYFILE   Path to key file, defaults to STDOUT
  -U, --user              Write to user key file (typically "${HOME}/.config/geneos/keyfile.aes")
  -b, --backup string     Backup existing keyfile with extension given (default ".old")
  -F, --force             Force overwriting an existing key file
  -S, --shared            Import the keyfile to component shared directories
      --update            Update shared keyfile on matching instances
```

## Examples

```bash
geneos aes new
geneos aes new -F ~/keyfile.aes
geneos aes new -S gateway

```

## SEE ALSO

* [geneos aes](geneos_aes.md)	 - AES256 Key File Operations
