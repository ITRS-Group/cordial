Create a new certificate, independent of instances

The `tls create` command differs from `tls new` by creating a certificate in the current working directory based on the Common Name given. You can use this command when you need to create certificates for manual configuration or transfer to another location.

By default a new root and intermediate certificate are created along with private keys in your user configuration directory unless they already exist and a certificate with a Common Name (CN) and a Subject Alternative Name (SAN) set to the system hostname is created, with the CN used as the base filename unless one exists. Any spaces in the CN are replaced with dashes (`-`) in the file name.

The Common Name can be set with the `--cname`/`-c` option and Subject Alternative Names can be added with the `--san`/`-s` option, which can be repeated as required.

If a file exists with the same name for the PEM file then it, and the corresponding key file with a `.key` extension, will only be overwritten if given the `--force`/`-F` option.
