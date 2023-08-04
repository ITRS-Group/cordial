# Import TLS certificates and keys

You can import certificates that you have generated or obtained externally along with private keys.

You can import two types of certificate; The first is an instance certificate for Geneos components acting as a server to use to validate themselves to client connections and the second is a signing certificate that can be used to create instance certificates. You can also import certificate chains, without private keys, to verify the chain of trust between two Geneos components.

You can only import either an instance certificate or a signing key at one time, but not both.

A certificate (in PEM format) can be imported for matching instances using `--cert`/`-c`. The file must also contain the unprotected private key or it can be given separately using the `--privkey`/`-k` option. This certificate and the unprotected private key will be applied to any matching instances but unlike other commands it will not be applied globally to all instances if no TYPE or NAME is given. If the certificate file contains other certificates then those will be treated like a certificate chain and written to an instance specific chain file and the instance parameter `certchain` set to the path to the file. If `certchain` is not set then the local Geneos installation-wide fallback certificate chain is used when available. Note that importing a new certificate without an embedded verification chain will leave any existing `certchain` parameter unchanged, which may be incorrect for the new certificate.

A signing certificate in PEM format can be imported using `--signing`/`-s`. The file must also contain the unprotected private key that can be used for signing insatnce certificates, or the key can be given separately using the `--privkey`/`-k` option. Without a valid signing certificate and matching key the commands `tls new` and `tls renew` will not work. The signing certificate is imported into the Geneos `tls` directory as `geneos.pem` and `geneos.key`. Only one signing certificate and matching key may be present at any time. If no chain is given with the `--chain`/`-C` option, see below, then all valid certificates, including the signer, with the IsCA flag set are used as a chain file and saved.

A certificate chain file containing multiple certificates in PEM format can be imported using the `--chain`/`-C` option. Only valid certificates (that satisfy the Basic Constraints extension validity check) and have the IsCA flag are written to the chain file.

If the private key is encrypted then it must be decrypted manually before import. Keys will be imported without encryption as they must be stored unprotected (except for file system permissions) in order that Geneos components to be able to use them.

