Import a certificate bundle. If no instance TYPE of NAMEs are provided then this is assumed to be a signer bundle containing the private key, signer certificate and root certificate in PEM format. If you specify a TYPE or NAMEs then the input is treated as an instance bundle instead, which must contains the private key, instance certificate and any intermediate certificates up to and including the trust root.

Note: Because a file name like `bundle.pem` can also be valid instance name it may be mistaken for one. To avoid this confusion it is recommended that you always give the file name using the `--file` (or `-f`) option, e.g. `geneos tls import netprobe myprobe --file bundle.pem`. The option can be repeated and takes glob patterns (quote them to stop your shell expanding them), and the resulting files are combined with any other arguments that are not instance names. Alternatively you can use a `./bundle.pem` style path to force it to be treated as a file name. Only one file can be imported at a time, except when using `--roots-only` described below.

In all cases the bundle must contain a verifiable trust chain from the most specific certificate (instance or signer) to the trust root. If this is not the case then the bundle will be rejected. The private key must also match the first certificate.

The input can be in either PEM or PFX/PKCS#12 format. If the input is in PFX/PKCS#12 format then you must provide the password to decrypt it using the `--password` option or you will be prompted for it.

Any trust root certificates in the bundle will be added to the local trust store (the `ca-bundle` files) if they are not already present.

### Importing only root certificates

If the input (without any instance TYPE or NAMEs) contains only root CA certificates, and no signer or intermediate certificates, then the root certificates are added to the `ca-bundle` files only. The local root CA (`rootCA.pem`) and signing certificate are not changed, no private key is required and any private key in the input is ignored.

To get the same behaviour from a file that also contains other certificates or private keys, use the `--roots-only` option (also accepted as `--root-only`). All certificates in the input that are not valid root CA certificates, and all private keys, are ignored and reported, and the import fails if no valid root CA certificates are found. The option can be used with PEM or PFX/PKCS#12 input. For PFX/PKCS#12 input the file can be either a private key and certificate bundle or a Java-style trust store containing only trusted certificates (as created by `keytool`). It cannot be combined with `--key` or with instance TYPE or NAMEs. This is the only mode that accepts more than one file, either by repeating `--file` or with a glob pattern, and the files are processed in order, stopping at the first error.

A root CA certificate must be self-signed, marked as a CA and be within its validity period. Any that are not are ignored.
