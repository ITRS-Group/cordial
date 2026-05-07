# `geneos host delete`

Delete the local configuration referring to a remote host.

## Usage

```text
geneos host delete [flags] NAME...
```

### Options

```text
  -F, --force           Delete instances without checking if disabled
  -R, --all             Recursively delete all instances on the host before removing the host config
  -S, --stop            Stop all instances on the host before deleting the local entry
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos host](geneos_host.md)	 - Remote Host Operations
