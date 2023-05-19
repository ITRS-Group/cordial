# `geneos tls sync`

Sync remote hosts certificate chain files

```text
geneos tls sync [flags]
```

## Details

Create a chain.pem file made up of the root and signing certificates and
then copy them to all remote hosts. This can then be used to verify
connections from components.

The root certificate is optional, b ut the signing certificate must
exist.

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - Manage certificates for secure connections
