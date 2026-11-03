Renew certificates for matching instances, or renew the local signing certificate with `--signing`. Instance renewal preserves the existing certificate's subject alternative names and defaults to a 365-day lifetime (`--expiry`/`-E`). Instances without an existing certificate are skipped; use [`geneos tls new`](geneos_tls_new.md) to create one.

The instance renewal modes are mutually exclusive:

- By default, write the renewed certificate and key to the active paths and update the instance configuration.
- `--prepare` (`-P`) writes the new certificate and key with a `-new` suffix without replacing the active files or restarting the instance. Check that the staged files are present before continuing.
- `--roll` (`-R`) promotes the staged files and keeps the previous active files with a `-old` suffix.
- `--unroll` (`-U`) restores the previous files from the `-old` suffix and leaves the renewed files staged with the `-new` suffix.

`--roll` and `--unroll` replace files but do not restart the instance. The running process continues to use its currently loaded certificate until it is restarted. This lets you stage certificates during active monitoring and schedule activation and restart for a maintenance window.

> [!IMPORTANT]
> Before using staged renewal on an instance that has not been migrated to the current TLS layout, migrate and validate it separately. `geneos tls renew` runs the TLS migration step before applying any renewal mode, and that step can update the instance files and configuration.

Use `--signing` to renew the local signing certificate instead of instance certificates. This requires the local root certificate and private key; `--expiry` does not apply to signing certificates.

### Staged renewal workflow

```bash
# During normal operation, stage the new certificate and key
geneos tls renew gateway mygateway --prepare

# During a maintenance window, activate them and restart the Gateway
geneos tls renew gateway mygateway --roll
geneos restart gateway mygateway

# If validation fails, restore the previous files and restart again
geneos tls renew gateway mygateway --unroll
geneos restart gateway mygateway
```

`--unroll` restores the saved files; restarting afterward makes the Gateway load them. Confirm the Gateway is healthy before ending the maintenance window.

### Renew the signing certificate

```bash
# Renew the local signing certificate
geneos tls renew --signing
```

