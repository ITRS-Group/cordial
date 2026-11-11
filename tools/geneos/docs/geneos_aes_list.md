# `geneos aes list`

The `aes list` command lists details of the key files referenced by matching instances.

If given the `--shared`/`-S` flag then the key files in the shared component directory are listed. This can be filtered by host with the `--host`/`-H` and/or by component `TYPE`.

The default output is human-readable table format. You can select CSV or JSON formats using the appropriate flags.

## Usage

```text
geneos aes list [flags] [TYPE] [NAME...]
```

### Options

```text
  -S, --shared[=true|false]       List shared key files
  -j, --json[=true|false]         Output JSON
  -i, --pretty[=true|false]       Output indented JSON
  -c, --csv[=true|false]          Output CSV
  -t, --toolkit[=true|false]      Output Toolkit formatted CSV
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string           profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos aes list gateway
geneos aes ls -S gateway -H localhost -c

```

## SEE ALSO

* [geneos aes](geneos_aes.md)	 - AES256 Key File Operations
