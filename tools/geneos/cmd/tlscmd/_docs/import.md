# `geneos tls import`

Import a certificate bundle. If no instance TYPE of NAMEs are provided then this is assumed to be a signer bundle containing the private key, signer certificate and root certificate in PEM format. If you specify a TYPE or NAMEs then the input is treated as an instance bundle instead, which must contains the private key, instance certificate and any intermediate certificates up to and including the trust root.

Note: Because a file name like `bundle.pem` can also be valid instance name you must use a `./bundle.pem` style path to force it to be treated as a file name when importing an instance bundle.

In all cases the bundle must contain a verifiable trust chain from the most specific certificate (instance or signer) to the trust root. If this is not the case then the bundle will be rejected. The private key must also match the first certificate.

The input can be in either PEM or PFX/PKCS#12 format. If the input is in PFX/PKCS#12 format then you must provide the password to decrypt it using the `--password` option or you will be prompted for it.

Any trust root certificates in the bundle will be added to the local trust store if they are not already present.
