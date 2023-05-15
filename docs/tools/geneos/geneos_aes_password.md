## geneos aes password

Encode a password using user's keyfile

### Synopsis


Encode a password using the user's keyfile. If no keyfile exists it
is created. Output is in `Expand` format.

User is prompted to enter the password (twice, for validation) unless
on of the flags is set.


```
geneos aes password [flags]
```

### Options

```
  -p, --password string   Password string to use
  -s, --source string     Source for password to use
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos aes](geneos_aes.md)	 - Manage Geneos compatible key files and encode/decode passwords

