# `geneos tls renew`

# `geneos tls renew`


## Usage

```text
geneos tls renew [TYPE] [NAME...] [flags]
```

### Options

```text
      --signing         Renew the signing certificate instead of instance certificates
  -E, --expiry int      Instance certificate expiry duration in days.
                        (No effect with --signing) (default 365)
  -P, --prepare         Prepare renewal without overwriting existing certificates
  -R, --roll            Roll previously prepared certificates and backup existing ones
  -U, --unroll          Unroll previously rolled certificates to restore backups
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
