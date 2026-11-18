# `geneos tls export`

Exports either the local signer private key, certificate and root certificate suitable for importing into another cordial `geneos` installation or, if you specify either component TYPE or instance NAMEs then the private key, instance certificate and other certificates up to and including the trust root for matching instances is output instead.

The default output is to the console but you can specify a file destination using the `--output`/`-o` option. For instance certificates a file destination can only be used if a single instance matches, as the certificates of every instance would otherwise be written to the same file.

## Usage

```text
geneos tls export [flags] [TYPE] [NAME...]
```

### Options

```text
  -o, --output FILE               Output to FILE, default to stdout
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/geneos/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string           profiles configuration (defaults are ~/.config/geneos/profiles.yaml, /etc/geneos/profiles.yaml)
```

## Examples

```bash
geneos tls export --out file.pem
geneos tls export gateway mygateway

```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
