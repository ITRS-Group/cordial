# `geneos tls import`

Import root and signing certificates

```text
geneos tls import
```

Import non-instance certificates. A root certificate is one where the
subject is the same as the issuer. All other certificates are imported
as signing certs. Only the last one, if multiple are given, is used.
Private keys must be supplied, either as individual files on in the
certificate files and cannot be password protected. Only certificates
with matching private keys are imported.

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - Manage certificates for secure connections
