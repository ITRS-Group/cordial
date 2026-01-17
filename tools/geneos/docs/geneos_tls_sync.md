# `geneos tls sync`

# `geneos tls sync`

The `geneos tls sync` command synchronises TLS certificates and keys from the local TLS environment to remote Geneos instances. This is useful when you have initialised or renewed certificates locally and need to distribute them to the instances. It also forces a rebuild of the local `ca-bundle.db` from `ca-bundle.pem` file which is a Java trust store containing the root CA certificate. The command will also remove any invalid or expired trust roots from the `ca-bundle.pem` file before rebuilding the trust store.

```text
geneos tls sync [flags]
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
