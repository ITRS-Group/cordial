# `geneos tls import`

# Import TLS certificates and keys

You can import certificates that have been generated externally along with their private keys and certificate chains.

You can import either an instance certificate, a signing certificate or a certificate chain.

## Signing Certificates

Without a valid signing certificate, validation chain and private key the commands `tls new` and `tls renew` will not work. Local signing certs and keys are created automatically when you initialize a Geneos host using `geneos init` or can be created on `geneos tls init` afterwards.

If you already have a signing certificate on another server but need to manage certificates without using the `geneos host` features then you can transfer the signing certificates to the remote server and then import it using this command or during set-up (`geneos init`) with the `--import-cert`/`-c` flag. You can also get the required certificates and key files from another Geneos host using the `geneos tls export` command.

All imported files must be in PEM format and private keys can be embedded in the same file or imported from a separate file.

Once copied to your remote server, the signing certificate and key can be imported using `--signer`/`-s`. The file must also contain the unprotected private key that can be used for signing instance certificates, or the key can be given separately using the `--key`/`-k` option.

The signing certificate is imported into the same file locations as above. Only one signing certificate (and chain) and matching key may be present at any time.

If no verification chain is given with the `--chain`/`-C` option then all valid certificates, including the signer, with the certificate authority flag set are used for the chain file.

## Instance Certificates

A certificate can be imported to matching instances using `--cert`/`-c`. The file can be either in PEM or PFX/PKCS#12 format - the latter requiring the extension to be either `.pfx` or `.p12` as the contents are not checked in advance of import.

If the certificate file is in PEM format it must contain the instance certificate to be imported and can also contain the unprotected private key or it can be given separately using the `--key`/`-k` option. If the PEM certificate file contains other certificates that are labelled as certificate authorities then those will be written to an instance specific verification chain file and the instance parameter `certchain` set to that file. If the private key is encrypted then it must be decrypted manually before import.

If the certificate file is in PFX/PKCS#12 format then it must contain both the instance certificate and the private key. If the PFX/PKCS#12 file is protected with a password then this can be given using the `--password`/`-p` option or you will be prompted to enter a password. Note that the private key is then imported unencrypted, as this is the required format for Geneos components to be able to use them. Other certificates in the PFX/PKCS#12 file that are labelled as certificate authorities will be written to an instance specific verification chain file and the instance parameter `certchain` set to that file.

The certificate and the key will be applied to any matching instances but, unlike many other commands, it will not be applied globally if no TYPE or NAME is given.

❗ Importing a certificate and key without a verification chain will leave any existing `certchain` parameter unchanged, which may be incorrect for the new certificate.

## Certificate Chain File

A certificate chain file containing multiple certificates in PEM format can be imported using the `--chain`/`-C` option. Only certificates that satisfy the Basic Constraints extension validity check and have the `IsCA` flag are written to the imported chain file, the rest are ignored.

```text
geneos tls import [flags] [TYPE] [NAME...]
```

### Options

```text
  -c, --instance-bundle string   Instance certificate bundle to import, PEM or PFX/PKCS#12 format
  -p, --password PLAINTEXT       Password for private key decryption, if needed, for pfx files
  -C, --signing-bundle string    Signing certificate bundle to import, PEM format
```

## Examples

```bash
$ geneos tls import netprobe localhost -c /path/to/file.pem
$ geneos tls import --signing-bundle /path/to/file.pem

```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
