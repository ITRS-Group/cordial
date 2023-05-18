# geneos tls renew

Renew instance certificates

```text
geneos tls renew [TYPE] [NAME...] [flags]
```

## Details

Renew instance certificates. All matching instances have a new
certificate issued using the current signing certificate but the
private key file is left unchanged if it exists.

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - Manage certificates for secure connections
