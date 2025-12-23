# `geneos tls migrate`

# `geneos tls migrate`

Migrate existing Geneos TLS certificates and related files to the updated layout and naming conventions.

Once migration has been performed, the older style files are no longer used by Geneos components and re-running this command will have no effect unless the `--unroll` option is used to restore previous backups.

```bash
geneos tls migrate --initial gateway mygateway
...
geneos tls migrate --final gateway mygateway
```

## Scenarios

1. Locally created certificates using the Geneos-specific root CA and signing certificates.

2. Certificates issued by an internal corporate PKI or external CA where the signing certificates are managed separately from the instance (leaf) certificates.

3. Mixed scenarios where some certificates are locally created and others are issued by an external CA.

4. External trust anchors managed outside of Geneos components, e.g. system-wide trust stores, are required to connect securely to certain external endpoints. For this scenario use the `geneos tls trust` command to add the required trust anchors to the Geneos-wide trust anchor file prior to migration.

## Actions

For each specified Geneos instance, one of the following actions is performed based on the provided flags:

### Initial Migration (`--initial`)

During the initial migration phase remote hosts acting as servers, e.g. Netprobes, may not have been migrated and so will only present the instance (leaf) certificate during TLS handshakes. This means that the instance being migrated needs to locally retain any intermediate signing certificates to validate the chain.

1. Existing certificate files, which only contain the instance ("leaf") certificate, have signing certificates appended to them to form a complete certificate chain. This means that the Geneos component will present the full chain during TLS handshakes, allowing clients to validate the entire chain of trust using just the root certificate in their trust store.

2. Existing chain files are decomposed. Any root certificate is added to the Geneos-wide trust anchor file if it is not already present. Intermediate certificates are left in the instance specific chain file as the peer, typically on a remote host, may only be presenting the leaf certificate during initial migration and a copy of the intermediates is required locally to validate the chain.

   A Java trust store file is also updated to include any new root certificates added to the Geneos-wide trust anchor file. This is used by Java-based Geneos components. This trust store uses the default Java `cacerts` password of `changeit`.

3. If the instance `use-chain` parameter is set to false, meaning the instance will not validate the remote peer then the above step is not required and the chain file is simply removed after any new root certificates have been added to the Geneos-wide trust anchor file.

### Final Migration (`--final`)

Once all remote hosts have been migrated, the final migration phase can be performed. During this phase, the instance being migrated can now validate the full chain presented by remote peers.

The instance specific chain file is no longer required as the full chain is presented by remote peers. The chain file is removed. If the `use-chain` parameter is set to false, meaning the instance will not validate the remote peer then no action is required.



```text
geneos tls migrate [TYPE] [NAME...] [flags]
```

### Options

```text
  -P, --prepare   Prepare migration without changing existing files
  -R, --roll      Roll previously prepared migrated files and backup existing ones
  -U, --unroll    Unroll previously rolled migrated files to earlier backups
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
