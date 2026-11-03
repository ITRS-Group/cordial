# `geneos tls new`

Create a leaf certificate and private key for each matching instance, signed by the local Geneos signing certificate. The command writes the files to the instance's configured TLS paths, updates its TLS parameters, and saves the instance configuration.

The local TLS environment must be initialised and its signing private key must be available. A matching instance that already has a valid certificate is left unchanged; use [`geneos tls renew`](geneos_tls_renew.md) to replace an existing certificate.

The certificate expiry defaults to 365 days. Set `--expiry` (`-E`) to change the lifetime.

Examples:

```bash
# Create certificates for one Gateway
geneos tls new gateway mygateway

# Create certificates for all Netprobes with a two-year lifetime
geneos tls new netprobe all --expiry 730
```


## Usage

```text
geneos tls new [TYPE] [NAME...] [flags]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -E, --expiry int        Certificate expiry duration in days (default 365)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
