Initialise the TLS environment by creating a self-signed root
certificate to act as a CA and a signing certificate signed by the root.
Any instances will have certificates created for them but configurations
will not be rebuilt.

To recreate the root and signing certificates and keys use the
`--force`/`-F` option.