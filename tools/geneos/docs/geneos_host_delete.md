# `geneos host delete`

Delete the local configuration referring to a remote host.

## Usage

```text
geneos host delete [flags] NAME...
```

### Options

```text
  -F, --force[=true|false]        Delete instances without checking if disabled
  -R, --all[=true|false]          Recursively delete all instances on the host before removing the host config
  -S, --stop[=true|false]         Stop all instances on the host before deleting the local entry
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string           profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos host](geneos_host.md)	 - Remote Host Operations
