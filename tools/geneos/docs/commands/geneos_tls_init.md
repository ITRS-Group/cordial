# `geneos tls init`

Initialise the TLS environment

```text
geneos tls init
```

Initialise the TLS environment by creating a self-signed root
certificate to act as a CA and a signing certificate signed by the root.
Any instances will have certificates created for them but configurations
will not be rebuilt.

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - Manage certificates for secure connections
