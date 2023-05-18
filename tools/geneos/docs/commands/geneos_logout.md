# geneos logout

Logout (remove credentials)

```text
geneos logout [flags] [NAME...]
```

## Details

The logout command removes the credentials for the names given. If no
names are set then the default credentials are removed.

If the `-A` options is given then all credentials are removed.

### Options

```text
  -A, --all   remove all credentials
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
