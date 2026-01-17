# `geneos tls import`

# `geneos tls import`

Import a certificate bundle. If no instance TYPE of NAMEs are provided then this is assumed to be a signer bundle containing the private key, signer certificate and root certificate in PEM format. If you specify a TYPE or NAMEs then the input is treated as an instance bundle instead, which must contains the private key, instance certificate and any intermediate certificates up to and including the trust root.

Note: Because a file name like `bundle.pem` can also be valid instance name you must use a `./bundle.pem` style path to force it to be treated as a file name when importing an instance bundle.

In all cases the bundle must contain a verifiable trust chain from the most specific certificate (instance or signer) to the trust root. If this is not the case then the bundle will be rejected. The private key must also match the first certificate.

The input can be in either PEM or PFX/PKCS#12 format. If the input is in PFX/PKCS#12 format then you must provide the password to decrypt it using the `--password` option or you will be prompted for it.

Any trust root certificates in the bundle will be added to the local trust store if they are not already present.

```text
geneos tls import [flags] [TYPE] [NAME...] [PATH]
```

### Options

```text
  -p, --password PLAINTEXT   Plaintext password for PFX/PKCS#12 file decryption.
                             You will be prompted if not supplied as an argument.
                             PFX/PKCS#12 files are identified by the .pfx or .p12
                             file extension and only supported for instance bundles
  -k, --key file             Private key file for certificate, PEM format only
```

## Examples

```bash
geneos tls import netprobe localhost /path/to/file.pem
geneos tls import /path/to/file.pem

```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
