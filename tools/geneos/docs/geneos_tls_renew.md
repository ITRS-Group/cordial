# `geneos tls renew`

# `geneos tls renew`


```text
geneos tls renew [TYPE] [NAME...] [flags]
```

### Options

```text
      --signer       Renew the signer certificate instead of instance certificates
  -E, --expiry int   Instance certificate expiry duration in days.
                     (No effect with --signer) (default 365)
  -P, --prepare      Prepare renewal without overwriting existing certificates
  -R, --roll         Roll previously prepared certificates and backup existing ones
  -U, --unroll       Unroll previously rolled certificates to restore backups
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
