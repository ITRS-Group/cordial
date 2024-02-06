# Import TLS certificates and keys

You can import certificates that you have previously generated or obtained externally along with their private keys.

You can import either an instance certificate or a signing certificate, and optionally a certificate chain.

All imported files must be in PEM format and private keys can be embedded in the same file or imported from a separate file.

## Instance Certificates

A certificate can be imported to matching instances using `--cert`/`-c`. The file must also contain the unprotected private key or it can be given separately using the `--privkey`/`-k` option. The certificate and the key will be applied to any matching instances but, unlike many other commands, it will not be applied globally if no TYPE or NAME is given.

If the certificate file contains other certificates that are labelled as certificate authorities then those will be written to an instance specific verification chain file and the instance parameter `certchain` set to that file.

If the private key is encrypted then it must be decrypted manually before import. Keys must be unencrypted as they must be stored unprotected (except for file system permissions) in order that Geneos components to be able to use them.

⚠️Warning: importing a certificate and key without a verification chain will leave any existing `certchain` parameter unchanged, which may be incorrect for the new certificate.

💡Note: While you can import certificates and keys for `webservers` instances, they will not be used directly as you will then need to import them into the Java truststore/keystore.

## Signing Certficates

Without a valid signing certificate and key the commands `tls new` and `tls renew` will not work.

If you have already created a signing certificate on anpother server but need to manage certificates on other servers without the SSH-based `host` feature then you can transfer the signing certificates to the remote server manually and then import it using this command. The default signing certificate and key files can be found at `${HOME}/.config/geneos/geneos.pem` and `${HOME}/.config/geneos/geneos.key`.

Once copied to your remote server, the signing certificate and key can be imported using `--signing`/`-s`. The file must also contain the unprotected private key that can be used for signing instance certificates, or the key can be given separately using the `--privkey`/`-k` option.

The signing certificate is imported into the same file locations as above. Only one signing certificate and matching key may be present at any time.

If no verification chain is given with the `--chain`/`-C` option then all valid certificates, including the signer, with the certificate authority flag set are used for the chain file.

## Certificate Chain File

A certificate chain file containing multiple certificates in PEM format can be imported using the `--chain`/`-C` option. Only certificates that satisfy the Basic Constraints extension validity check and have the `IsCA` flag are written to the imported chain file, the rest are ignored. If you import both instance certificates and a chain file they are imported independently and while the `--chain` /`-C` chain will be written to the global file, instances will still have specific chains imported and the `certchain` parameters set to use those.
