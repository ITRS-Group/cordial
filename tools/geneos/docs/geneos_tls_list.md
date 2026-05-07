# `geneos tls list`

List certificates and their details. The root and signing certs are only shown if the `--all`/`-a` flag is given. A list with more details can be seen with the `--long`/`-l` flag, otherwise options are the same as for the main ls command.

Certificates for each instance are validated and the "Valid" column or field contains the boolean result. An instance certificate is valid if **all** the following are true:

* The file path from the instance `certificate` parameter is readable, is in PEM format and can be parsed as an x509 certificate
* The file path from the instance `privatekey` parameter is readable, is in PEM format and matches the certificate above
* The certificate is inside it's validity period
* The certificate can be verified against the trust chain from one of:
    * the file path from the instance `certchain` parameter
    * the installation global `tls/geneos-chain.pem` file
    * the system certificate pool
* The certificate also conforms to other checks done by <https://pkg.go.dev/crypto/x509#Certificate.Verify>

The Common Name (`CN`) and the Subject Alternative Names (`SAN`) values in the certificate are not otherwise checked as Geneos does not use these.

## Usage

```text
geneos tls list [flags] [TYPE] [NAME...]
```

### Options

```text
  -a, --all             Show all certs, including root and signing certs
  -l, --long            Long output
  -j, --json            Output JSON
  -i, --pretty          Output indented JSON
  -c, --csv             Output CSV
  -t, --toolkit         Output Toolkit formatted CSV
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
