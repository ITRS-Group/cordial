## geneos aes set

Set (and import) keyfile for instances

### Synopsis

Set keyfile for matching instances. Either a path or URL to a
keyfile or the CRC of an existing keyfile in the component's shared
directory must be given. If a path or URL is given then the keyfile
is saved to the component shared directories and the configuration
set to reference that path. Unless the '-N' flag is given any
existing keyfile path is copied to a 'prevkeyfile' setting to support
key file updating in Geneos GA6.x.

If the '-C' flag is used and it identifies an existing keyfile in the
component keyfile directory then that is used for matching instances.

The argument given with the '-k' flag can be a local file (including
a prefix of '~/' to represent the home directory), a URL or a dash
'-' for STDIN.

Currently only Gateways and Netprobes (and SANs) are supported.

```
geneos aes set [-N] [-k FILE|URL|-] [-C CRC] [TYPE] [NAME...]
```

### Options

```
  -C, --crc string       CRC of keyfile to use.
  -h, --help             help for set
  -k, --keyfile string   Keyfile to use (default "/home/peter/.config/geneos/keyfile.aes")
  -N, --noroll           Do not roll any existing keyfile to previous keyfile setting
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos aes](geneos_aes.md)	 - Manage Gateway AES key files

