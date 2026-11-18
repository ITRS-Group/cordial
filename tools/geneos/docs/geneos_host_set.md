# `geneos host set`

Set options on remote host configurations.

## Usage

```text
geneos host set [flags] [NAME...] [KEY=VALUE...]
```

### Options

```text
  -p, --prompt[=true|false]       Prompt for password
  -P, --password SECRET           password
  -k, --keyfile KEYFILE           Keyfile
  -i, --privatekey PATH           Private key file
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/geneos/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string           profiles configuration (defaults are ~/.config/geneos/profiles.yaml, /etc/geneos/profiles.yaml)
```

## SEE ALSO

* [geneos host](geneos_host.md)	 - Remote Host Operations
