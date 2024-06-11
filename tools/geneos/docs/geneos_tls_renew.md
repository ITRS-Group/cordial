# `geneos tls renew`

# `geneos tls renew`

Renew instance certificates. All matching instances have a new certificate issued using the current signing certificate but the private key file is left unchanged if it exists, or created if it does not.

Use the `--days`/`-D` flag to set the expiry of the certificate, in 24 hour days (ignoring time-zone changes) from now. Certificates are created with a valid-before time of one minute before running the command, to allow for clock differences and latency of command execution.

```text
geneos tls renew [TYPE] [NAME...] [flags]
```

### Options

```text
  -D, --days int   Certificate duration in days (default 365)
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
