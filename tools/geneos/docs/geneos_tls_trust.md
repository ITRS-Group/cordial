# `geneos tls trust`

Import trusted certificates

```text
geneos tls trust [flags] [PATH...]
```

### Options

```text
  -r, --remove CN/SHA1/SHA256   Remove trusted certificates instead of adding them
                                The argument is either the certificate's CommonName or SHA1 or SHA256 fingerprint
                                This flag can be specified multiple times to remove multiple certificates
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
