## geneos aes decode

Decode a Geneos-format secure password

### Synopsis


Decode a Geneos-format secure password using the keyfile(s) given.

If no keyfiles are provided then all matching instances are checked
for configured keyfiles and each one tried or the default keyfile
paths are tried. An error is only returned if all attempts to decode
fail. The cipertext may contain the optional prefix `+encs+`. If both
`-P` and `-s` options are given then the `-P` argument is used. To
read a ciphertext from STDIN use `-s -`.

If an `expandable` string is given with the `-e` option it must be of
the form `${enc:...}` (be careful to single-quote this string when
using a shell) and is then decoded using the keyfile and ciphertext
in the value. All other flags and arguments are ignored.


```
geneos aes decode [flags] [TYPE] [NAME...]
```

### Options

```
  -e, --expand string     A string in ExpandString format (including '${...}') to decode
  -k, --keyfile string    Main AES key file to use (default "/home/peter/.config/geneos/keyfile.aes")
  -v, --previous string   Previous AES key file to use (default "/home/peter/.config/geneos/prevkeyfile.aes")
  -p, --password string   Password to decode
  -s, --source string     Source for password to use
  -h, --help              help for decode
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos aes](geneos_aes.md)	 - Manage Gateway AES key files

