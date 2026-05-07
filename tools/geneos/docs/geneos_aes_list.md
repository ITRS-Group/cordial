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
  -S, --shared          List shared key files
  -j, --json            Output JSON
  -i, --pretty          Output indented JSON
  -c, --csv             Output CSV
  -t, --toolkit         Output Toolkit formatted CSV
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## Examples

```bash
geneos aes list gateway
geneos aes ls -S gateway -H localhost -c

```

## SEE ALSO

* [geneos aes](geneos_aes.md)	 - AES256 Key File Operations
