# `geneos aes import`

Import key files to the `TYPE/keyfiles` directory in each matching component TYPE shared directory.

A key file is provided with the `--keyfile`/`-k` option. The default is to read from STDIN. You can import from a local file (a path prefixed with `~/` is treated as relative to your home directory), a remote URL or STDIN.

The key file is copied from the supplied file to a file with the base-name of its 8-hexadecimal digit checksum to distinguish it from other key files. In all examples the CRC is shown as `DEADBEEF` in honour of many generations of previous UNIX documentation. There is a very small chance of a checksum clash.

The shared directory for each component is one level above instance directories and has a `_shared` suffix. The convention is to use this path for Geneos instances to share common configurations and resources. e.g. for a Gateway the path would be `.../gateway/gateway_shared/keyfiles` where instance directories would be `.../gateway/gateways/NAME`

If a `TYPE` is given then the key is only imported for that component, otherwise the key file is imported to all components that are known to support key files. Currently only Gateways and Netprobes (including SANs) are supported.

Key files are imported to all configured hosts unless `--host`/`-H` is used to limit to a specific host.

Instance names can be given to indirectly identify the component type.

```text
geneos aes import [flags] [TYPE] [NAME...]
```

### Options

```text
  -k, --keyfile string   Path to key-file (default "-")
```

## Examples

```bash
# import keyfile.aes to GENEOS/gateway/gateway_shared/DEADBEEF.aes
geneos aes import --keyfile ~/keyfile.aes gateway

```

## SEE ALSO

* [geneos aes](geneos_aes.md)	 - AES256 Key File Operations
