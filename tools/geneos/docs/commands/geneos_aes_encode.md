## geneos aes encode

Encode plaintext to a Geneos AES256 password using a key file

### Synopsis


Encode plaintext to a Geneos AES256 format password using a key file.

By default the user is prompted to enter a password but can provide a
string or URL with the `-p` option. If TYPE and NAME are given then
the key files are checked for those instances. If multiple instances
match then the given password is encoded for each keyfile found.

It is important to note that no whitespace is trimmed from the
plaintext. This can have unexpected results if you do something like
this:

$ echo "test" ` geneos aes encode -s -

rather then this:

$ echo -n "test" ` geneos aes encode -s -

	

```
geneos aes encode [flags] [TYPE] [NAME...]
```

### Options

```
  -e, --expandable        Output in 'expandable' format
  -k, --keyfile KEYFILE   Path to keyfile
  -p, --password string   Plaintext password
  -s, --source string     Alternative source for plaintext password
  -o, --once              Only prompt for password once, do not verify. Normally use '-s -' for stdin
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos aes](geneos_aes.md)	 - Manage Geneos compatible key files and encode/decode passwords

