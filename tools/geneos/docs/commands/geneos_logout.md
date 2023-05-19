# `geneos logout`

Logout (remove credentials)

```text
geneos logout [flags] [DOMAIN...]
```

The logout command removes the credentials for the `DOMAIN` given. If no
names are set then the default credentials (`itrsgroup.com`) are
removed.

If the `-A` options is given then all credentials are removed, but the
underlying file is not deleted.

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

* [geneos](geneos.md)	 - Take control of your Geneos environments
