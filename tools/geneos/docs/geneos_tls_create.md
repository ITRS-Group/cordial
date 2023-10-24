# `geneos tls create`

Create a new certificate, independent of instances

The `tls create` command differs from `tls new` by creating a certificate in the current working directory based on the common name, and not for instances directly. You can use this command when you need to create certificates for manual configuration or transfer to another location.

By default a new root and intermediate certificate are created along with private keys in your user configuration directory unless they already exist and a certificate with a Common Name (CN) and a Subject Alternative Name (SAN) set to the system hostname is created, with the CN used as the base filename unless one exists. Any spaces in the CN are replaced with dashes (`-`)

The Common Name can be set with the `--cname`/`-c` option and Subject Alternative Names can be added with the `--san`/`-s` option, which can be repeated as required.

If a file exists with the resulting CN with a `.pem` extension then it, and the matching key file with a `.key` extension, will only be overwritten if given the `--force`/`-F` option.

```text
geneos tls create [flags]
```

### Options

```text
  -c, --cname string   Common Name for certificate. Defaults to hostname
  -F, --force          Force overwrite existing certificate (but not root and intermediate)
  -s, --san SAN        Subject-Alternative-Name (repeat for each one required). Defaults to hostname if none given
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
