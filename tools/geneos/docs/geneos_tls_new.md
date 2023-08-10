# `geneos tls new`

Create new certificates

```text
geneos tls new [TYPE] [NAME...] [flags]
```

The `tls new` command creates new certificates for matching instances. It overwrites existing certificates.

To create new certificates there must be a valid signing certificate and matching private key. These can be created using the `tls init` command or you can import them using `tls import`.

The `tls new` command differs from `tls create` as the latter creates new certificates in your current directory for later use, while this command creates certificates for matching instances and sets the Common Name based on the component type and name for simple identification.

⚠ Warning: While you can create certificates and keys for `webservers` they will not be used directly as you need to manually import them into the configured truststore/keystore.

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
