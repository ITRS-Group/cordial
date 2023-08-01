# Import TLS certificates and keys

You can import certificates that you have generated or obtained externally along with private keys.

You can import two types of certificate; The first is an instance certificate for Geneos components acting as a server to use to validate themselves to client connections and the second is a signing certificate that can be used to create instance certificates. You can also import certificate chains, without private keys, to verify the chain of trust between two Geneos components.

You can only import either an instance certificate or a signing key at one time, but not both.

An instance certificate in PEM format can be imported using `--cert`/`-c`. The file may contain the private key or it can be given separately using the `--privkey`/`-k` option. This certificate and the unencrypted private key will be applied to any matching instances but unlike other commands it will not be applied to all instances if no TYPE or NAME is given. If the certificate file contains other certificates then those will be treated like a certificate chain, see below.

A signing certificate in PEM format can be imported using `--signing`/`-s`. The file may also contain the private key that can be used for signing certificates created, or the key can be given separately using the `--privkey`/`-k` option. Without a valid signing certificate and matching key the commands `tls new` and `tls renew` will not work.

A certificate chain file containing multiple certificates in PEM format can be imported using the `--chain`/`-C` option. Any leaf certificate, that is one that does not have a Basic Constraint marking it as a certificate authority, is extracted and will be used as an instance certificate while the remaining certificates are written to a chain file in PEM format that can be sed with the `-ssl-certificate-chain` option on Geneos components. Any private keys apart from one for the leaf certificate will be ignored and the certificates in the imported chain cannot be used for signing new certificates.

If the private key is encrypted then it must be decrypted manually before import. Keys will be imported without encryption as they must be stored unprotected (except for file system permissions) in order that Geneos components to be able to use them.

No validation is done by the `tls import` command on the compatibility of a signing certificate, a chain file and a certificate file. You can use the `tls ls` command to validate installed certificates after import.
