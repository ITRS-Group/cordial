By default the `new` command creates new certificates for matching
instances. It overwrites existing certificates.

Using the `--named`/`-n` option will instead create a new certificate
and key pair for the CN (Common Name) cn and name the file for the CN
with spaces replaced by dashes. The Subject Alternative Name for the
certificate is set from the machine hostname but can be overridden using
the global `--hostname`/`-H` option. The certificate and key are
written to a .pem and .key file in the current directory unless you use
the `--dir`/`-D` option.
