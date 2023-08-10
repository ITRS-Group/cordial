# `geneos tls renew`

Renew instance certificates

```text
geneos tls renew [TYPE] [NAME...] [flags]
```

Renew instance certificates. All matching instances have a new certificate issued using the current signing certificate but the private key file is left unchanged if it exists.

⚠ Warning: While you can renew certificates and keys for `webservers` they will not be used directly as you need to manually import them into the configured truststore/keystore.

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
