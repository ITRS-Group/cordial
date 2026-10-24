# `geneos includes extract`

Extract includes from the geneos binary and place them in a versioned directory under the gateway directory.

Functional include files are extracted under `${GENEOS}/gateway/includes/<version>`. These are then linked back to the parent directory using symlinks to the chosen version. These files are all prefixes `itrs.` and are considered managed by the system.

Configuration templates are extracted under `${GENEOS}/gateway/include/config` but are never overwritten as it is assumed that these are edited by the user. These files are all prefixed `config.`.

## Usage

```text
geneos includes extract [flags]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -F, --force             Overwrite existing files
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos includes](geneos_includes.md)	 - Manage includes
