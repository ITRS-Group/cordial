## geneos logout

Logout (remove credentials)

### Synopsis


The logout command removes the credentials for the names given. If no
names are set then the default credentials are removed.

If the `-A` options is given then all credentials are removed.


```
geneos logout [flags] [NAME...]
```

### Options

```
  -A, --all   remove all credentials
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

