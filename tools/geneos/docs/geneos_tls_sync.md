# `geneos tls sync`

Sync remote hosts certificate chain files

```text
geneos tls sync [flags]
```

Create a chain.pem file made up of the root and signing certificates and
then copy them to all remote hosts. This can then be used to verify
connections from components.

The root certificate is optional, b ut the signing certificate must
exist.

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
