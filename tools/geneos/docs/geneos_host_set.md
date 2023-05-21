# `geneos host set`

Set host configuration value

```text
geneos host set [flags] [NAME...] [KEY=VALUE...]
```

Set options on remote host configurations.

### Options

```text
  -p, --prompt               Prompt for password
  -P, --password PLAINTEXT   password
  -k, --keyfile KEYFILE      Keyfile
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos host](geneos_host.md)	 - Manage remote host settings
