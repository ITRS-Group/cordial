# `geneos tls init`

Initialise the TLS environment by creating a self-signed root certificate to act as a CA and a signing certificate signed by the root. Any instances will have certificates created for them but configurations will not be rebuilt.

To recreate the root and signing certificates and keys use the `--force`/`-F` option.

All certificates are created with corresponding private keys. These keys are in ECDH format by default but this can be overridden using the `--keytype`/`-K` option which supports the following formats: "ecdh", "ecdsa", "ed25529" and "rsa". Once set for the root CA, all subsequent certificates will be created using the same key type. You should avoid "ed25519" as this is not supported by normal web broswers and will make it impossible to use the ORB diagnostic interfaces of Geneos.
```text
geneos tls init
```

### Options

```text
  -K, --keytype string   Key type for root, one of ecdh, ecdsa, ec15529 or rsa (default "ecdh")
  -F, --force            Overwrite any existing certificates
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
