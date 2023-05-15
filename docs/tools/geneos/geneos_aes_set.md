## geneos aes set

Set keyfile for instances

### Synopsis


Set a keyfile for matching instances. The keyfile is saved to each
matching component shared directory and the configuration set to
that path.

The keyfile can be given as either an existing CRC (without file
extension) or as a path or URL. If neither `-C` or `-k` are given
then the user's default keyfile is used, if found.

If the `-C` flag is given and it identifies an existing keyfile in
the component shared directory then that is used for matching
instances. When TYPE is not given, the keyfile will also be copied to
the shared directories of other component types if not already
present.

The `-k` flag value can be a local file (including a prefix of `~/`
to represent the home directory), a URL or a dash `-` for STDIN. The
given keyfile is evaluated and its CRC32 checksum checked against
existing keyfiles in the matching component shared directories. The
keyfile is only saved if one with the same checksum does not already
exist. 

Any existing `keyfile` path is copied to a `prevkeyfile` setting,
unless the `-N` option if given, to support key file updating in
Geneos GA6.x.

Currently only Gateways and Netprobes (and SANs) are supported.

Only local keyfiles, unless given as a URL, can be copied to remote
hosts, not visa versa. Referencing a keyfile by CRC on a remote host
will not result in that file being copies to other hosts.


```
geneos aes set [flags] [TYPE] [NAME...]
```

### Options

```
  -C, --crc string        CRC of existing component shared keyfile to use
  -k, --keyfile KEYFILE   Keyfile to import and use (default /home/peter/.config/geneos/keyfile.aes)
  -N, --noroll            Do not roll any existing keyfile to previous keyfile setting
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos aes](geneos_aes.md)	 - Manage Geneos compatible key files and encode/decode passwords

