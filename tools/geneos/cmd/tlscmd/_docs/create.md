# `geneos tls create`

Creates a new TLS certificate bundle including a private key and trust chain. This can either be an instance bundle, suitable for use with remote Geneos components, for example Windows Netprobes, or a signing certificate bundle for use on other systems where `geneos` will be used.

By default the command will create a file containing a private key, certificate and trust chain in the current working directory using the local hostname as the Common Name (`CN`). You can specify a different Common Name using the `--cname`/`-c` option. If you want to include spaces please remember to quote the name. The resulting file has spaces in the Common Name replaced with dashes.

To change to output directory use the `--dest`/`-D` option. The name of the file is always derived from the common name, with spaces replaced by dashes, and a `.pem` extension. The only exception to this is when using an output destination of `-` which will write the output to standard output instead of a file.

The default expiry period is 365 days (from one minute in the past - this is to allow some overlap with system clock issues) unless you use the `--expiry`/`-E` option. This option is ignored when creating a signing certificate with the `--signer`/`-s` option, as signing certificates always have a fixed validity period of 5 years.

If the Geneos installation has not been initialised or the TLS sub-system not been initialised then an error will be returned. With the `--force`/`-F` flags a new root and intermediate certificate are created along with private keys in your user configuration directory unless they already exist. Beware that initialising the TLS subsystem will also result in any new instances you create having certificates automatically created and various options set to trigger the use of TLS by default.

You can add Subject Alternative Names (SANs) to the certificate using the `--san-dns`/`-s`, `--san-ip`/`-i`, `--san-email`/`-e` and `--san-url`/`-u` options respectively. These options can be repeated as required to add multiple SANs of each type. SANs are ignored when creating a signing certificate with the `--signer`/`-s` option. If no DNS SANs are specified then the Common Name will be added as a DNS SAN automatically.

If a certificate already exists for the specified Common Name then an error will be returned unless you use the `--force`/`-F` option to overwrite it.
