# `geneos tls export`

The `tls export` command gathers and outputs the local Geneos signing certificate and private key and the root CA certificate (but not the private key) as a single PEM file.

By default the PEM formatted set of certificates and key is to the console but you can write them to a file using the `--output FILE`/`-o FILE` option. The file is created with 0600 permissions.

To not include the root CA certificate, which may be valid in some limited cases, use the `--no-root`/`-N` option.

The resulting PEM data can be imported into another Geneos instance through one of the `geneos deploy --import-cert`, `geneos init --import-cert` or `geneos tls import --signer` commands.

```text
geneos tls export [flags]
```

### Options

```text
  -o, --output string   Output destination, default to stdout
  -N, --no-root         Do not include the root CA certificate
```

## Examples

```bash
# export 
$ geneos tls export --output file.pem

```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
