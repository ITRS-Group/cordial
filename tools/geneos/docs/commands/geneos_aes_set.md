## geneos aes set

Set active keyfile for instances

### Synopsis


Set a key file for matching instances. The key file is saved to each
matching component's (default: all) shared directory and the
configuration set to that path.

The keyfile can be given as an existing CRC (with or without `.aes`
file extension) with the `--crc`/`-c` option or as a file path (which
can be prefixed `~/` for the user's home directory) or a URL with
`--keyfile`/`-k`. If neither option is given then the user's default
keyfile is used, if it exists.

If the `--crc`/`-c` flag is given and it matches an existing
keyfile in the component shared directory then that is used for
matching instances. When TYPE is not given, the keyfile will also be
copied to the shared directories of other component types if not
already present.

The `--keyfile`/`-k` flag value can be a local file (including a
prefix of `~/` to represent the home directory), a URL or a dash `-`
for STDIN. The given keyfile is evaluated and its CRC32 checksum
checked against existing keyfiles in the matching component shared
directories. The keyfile is only saved if one with the same checksum
does not already exist. 

For each instance any existing `keyfile` path is copied to a
`prevkeyfile` setting, unless the `--noroll`/`-N` option if given, to
support key file updating in Geneos GA6 and above.

Key files are only set on components that support them.

Only local keyfiles, unless given as a URL, can be copied to remote
hosts, not visa versa. Referencing a keyfile by CRC on a remote host
will not result in that file being copies to other hosts.





```
geneos aes set [flags] [TYPE] [NAME...]
```

### Options

```
  -c, --crc string        CRC of existing component shared keyfile to use (extension optional)
  -k, --keyfile KEYFILE   Key file to import and use (default /home/peter/.config/geneos/keyfile.aes)
  -N, --noroll            Do not roll any existing keyfile to previous keyfile setting
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

### SEE ALSO

* [geneos aes](geneos_aes.md)	 - Manage Geneos compatible key files and encode/decode passwords

