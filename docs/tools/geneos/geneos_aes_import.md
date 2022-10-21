## geneos aes import

Import shared keyfiles for components

### Synopsis


Import keyfiles to component shared directories.

The argument given with the '-k' flag can be a local file (including
a prefix of '~/' to represent the home directory), a URL or a dash
'-' for STDIN. If no '-k' flag is given then the user's default
keyfile is imported.

If a TYPE is given then the key is only imported to that component
type, otherwise the keyfile is imported to all supported components.
Currently only Gateways and Netprobes (and SANs) are supported.

Keyfiles are imported to all configured hosts unless '-H' is used to
limit to a specific host.

Instance names can be given to indirectly identify the component
type.


```
geneos aes import [flags] [TYPE] [NAME...]
```

### Options

```
  -k, --keyfile string   Keyfile to use (default "/home/peter/.config/geneos/keyfile.aes")
  -H, --host string      Import only to named host, default is all
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos aes](geneos_aes.md)	 - Manage Gateway AES key files

