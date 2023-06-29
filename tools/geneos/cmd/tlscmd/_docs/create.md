Create a new certificate

By default a new root and intermediate certificate are created along with private keys in your user configuration directory unless they already exist and a certificate with a Common Name (CN) and a Subject Alternative Name (SAN) set to the system hostname is created, with the CN used as the base filename unless one exists. Any spaces in the CN are replaced with dashes (`-`)

The Common Name can be set with the `--cname`/`-c` option and Subject Alternative Names can be added with the `--san`/`-s` option, which can be repeated as required.

If a file exists with the resulting CN with a `.pem` extension then it, and the matching key file with a `.key` extension, will only be overwritten if given the `--force`/`-F` option.
