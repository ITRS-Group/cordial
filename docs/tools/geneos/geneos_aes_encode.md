## geneos aes encode

Encode a password using a Geneos AES file

### Synopsis


Encode a password (or any other string) using the keyfile for a
Geneos Gateway. By default the user is prompted to enter a password
but can provide a string or URL with the -p option. If TYPE and NAME
are given then the key files are checked for those instances. If
multiple instances match then the given password is encoded for each
keyfile found.


```
geneos aes encode [flags] [TYPE] [NAME...]
```

### Options

```
  -k, --keyfile string    Main AES key file to use (default "/home/peter/.config/geneos/keyfile.aes")
  -p, --password string   Password string to use
  -s, --source string     Source for password to use
  -e, --expandable        Output in ExpandString format
  -o, --once              One prompt for password once. For scripts injecting passwords on stdin
  -h, --help              help for encode
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos aes](geneos_aes.md)	 - Manage Gateway AES key files

