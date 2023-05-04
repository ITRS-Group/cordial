## geneos aes encode

Encode a password using a Geneos compatible keyfile

### Synopsis


Encode a password (or any other string) using a Geneos compatible keyfile.

By default the user is prompted to enter a password but can provide a
string or URL with the `-p` option. If TYPE and NAME are given then
the key files are checked for those instances. If multiple instances
match then the given password is encoded for each keyfile found.


```
geneos aes encode [flags] [TYPE] [NAME...]
```

### Options

```
  -k, --keyfile string    Specific AES key file to use. Ignores matching instances
  -p, --password string   Password string to use
  -s, --source string     Source for password to use
  -e, --expandable        Output in ExpandString format
  -o, --once              Only prompt for password once. For scripts injecting passwords on stdin
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos aes](geneos_aes.md)	 - Manage Geneos compatible key files and encode/decode passwords

