# `geneos tls new`

# `geneos tls new`

The `tls new` command creates new certificates for matching instances. It does not overwrite existing certificates, use `tls renew` to do that.

To create new certificates there must be a valid signing certificate and private key. These can be created using the `tls init` command or you can import them using `tls import --signer`.

Use the `--days`/`-D` flag to set the expiry of the certificate, in 24 hour days (ignoring time-zone changes) from now. Certificates are created with a valid-before time of one minute before running the command, to allow for clock differences and latency of command execution.

The `tls new` command differs from `tls create` as the latter creates new certificates in your current directory for later use, while this command creates certificates for matching instances and sets the Common Name based on the component type and name for simple identification. Geneos components do not check the Common Name or related field in the certificate.

💡The command will skip any instances that have an existing, valid certificate file and key. To overwrite existing certificates and keys use the `tls renew` command.

⚠️ Warning: While you can create certificates and keys for `webservers`, they will not be used directly as you need to import them into the Java truststore/keystore.

```text
geneos tls new [TYPE] [NAME...] [flags]
```

### Options

```text
  -D, --days int   Certificate duration in days (default 365)
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
