# Import TLS certificates and keys

You can import certificates that you have generated or obtained externally, and their private keys.

A signing certificate in PEM format can be imported using `--signing`/`-S`. The file may also contain the private key that can be used for signing certificates created, or the key can be given separately using the `--signingkey`/`-K` option. Without a valid signing certificate the commands `tls new` and `tls renew` will not work.

An instance certificate in PEM format can be imported using `--cert`/`-c`. The file may contain the private key or it can be given separately using the `--privkey`/`-k` option. This certificate and the unencrypted private key will be applied to any matching instances. If the certificate file contains other certificates then it will be treated like a certificate chain, below.

A certificate chain file containing multiple certificates in PEM format can be imported using the `--chain`/`-C` option. This can only be used when not also importing a signing certificates. Any leaf certificate, that is one that is not labelled as a certificate authority, is extracted and will be used as an instance certificate while the remaining certificates are written to a chain file in PEM format that can be sed with the `-ssl-certificate-chain` option on Geneos components. Any private keys apart from one for the leaf certificate will be discarded. The imported chain cannot be used for signing new certificates.

If keys are encrypted then they must be decrypted manually before import. Each key will be imported without encryption as they must be stored unprotected (except for file system permissions) in order that Geneos components to be able to load them.

No validation is done by the `tls import` command on the compatibility of a signing certificate, a chain file and a certificate file. You can use the `tls ls` command to validate installed certificates after import.
