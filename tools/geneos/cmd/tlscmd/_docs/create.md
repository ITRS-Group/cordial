# `geneos tls create`

Create a standalone certificate and private key, with optional bundle including parent certificates.

⚠ The `tls create` command differs from `tls new` and `tls renew` by creating a certificate in the current working directory based on the Common Name given. You can use this command when you need to create certificates for manual configuration or transfer to another location. You will probably want to use `tls new` and `tls renew` for most Geneos certificate management.

Without other options `tls create` will create certificate and private key files (using extensions `.pem` and `.key` respectively) in the current directory, using an existing Geneos signing certificate, for use with Geneos components. The file names will be based on the certificate Common Name (`CN`), which will default to the local hostname.

You can set the destination directory using the `--dest` option.

To create a certificate bundle (sometimes referred to as a `fullchain.pem` file) use the `--bundle`/`-b` flag. This created a single `.pem` file using the same options as above unless the `--dest` directory is given as a dash (`-`) in which case the PEM formatted bundle is output to the console.

You can set you own Common Name using the `--cname`/`-c` option. If you want to include spaces please remember to quote the name. The resulting files have spaces in the Common Name replaced with dashes.

The default expiry period is 365 days from one minute in the past - this is to allow some overlap with system clock issues - unless you use the `--days`/`-D` option.

If the Geneos installation has not been initialised or the TLS sub-system not been initialised then an error will be returned.

With the `--force`/`-F` flags a new root and intermediate certificate are created along with private keys in your user configuration directory unless they already exist. The act of initialising the TLS subsystem will result in any new Geneos component instances you create having certificates automatically created and various options set to trigger the use of TLS by default - which may not be what you expect, so please beware.

To add Subject Alternative Names (`SANs`) use the `--san`/`-s` option and repeat as often as required.

If a file exists with the same name for the PEM file then it, and the corresponding key file with a `.key` extension, will only be overwritten if given the `--force`/`-F` option.
