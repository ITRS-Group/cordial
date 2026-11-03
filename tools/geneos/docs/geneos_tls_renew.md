# `geneos tls renew`

Renew certificates for matching instances, or renew the local signing certificate with `--signing`. Instance renewal preserves the existing certificate's subject alternative names and defaults to a 365-day lifetime (`--expiry`/`-E`). Instances without an existing certificate are skipped; use [`geneos tls new`](geneos_tls_new.md) to create one.

The instance renewal modes are mutually exclusive:

- By default, write the renewed certificate and key to the active paths and update the instance configuration.
- `--prepare` (`-P`) writes the new certificate and key with a `-new` suffix without replacing the active files. Check that the staged files are present before continuing.
- `--roll` (`-R`) promotes the staged files and keeps the previous active files with a `-old` suffix.
- `--unroll` (`-U`) restores the previous files from the `-old` suffix and leaves the renewed files staged with the `-new` suffix.

Use `--signing` to renew the local signing certificate instead of instance certificates. This requires the local root certificate and private key; `--expiry` does not apply to signing certificates.

Examples:

```bash
# Stage renewed certificates for a Gateway
geneos tls renew gateway mygateway --prepare

# Promote the staged files, or restore the previous files if needed
geneos tls renew gateway mygateway --roll
geneos tls renew gateway mygateway --unroll

# Renew the local signing certificate
geneos tls renew --signing
```


## Usage

```text
geneos tls renew [TYPE] [NAME...] [flags]
```

### Options

```text
      --signing           Renew the signing certificate instead of instance certificates
  -E, --expiry int        Instance certificate expiry duration in days.
                          (No effect with --signing) (default 365)
  -P, --prepare           Prepare renewal without overwriting existing certificates
  -R, --roll              Roll previously prepared certificates and backup existing ones
  -U, --unroll            Unroll previously rolled certificates to restore backups
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
