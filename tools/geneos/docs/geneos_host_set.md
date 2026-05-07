# `geneos host set`

Set options on remote host configurations.

## Usage

```text
geneos host set [flags] [NAME...] [KEY=VALUE...]
```

### Options

```text
  -p, --prompt            Prompt for password
  -P, --password SECRET   password
  -k, --keyfile KEYFILE   Keyfile
  -i, --privatekey PATH   Private key file
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos host](geneos_host.md)	 - Remote Host Operations
