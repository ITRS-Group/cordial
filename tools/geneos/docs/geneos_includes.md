# `geneos includes`

Geneos gateway includes are reusable pieces of configuration or code that can be incorporated into various parts of the Geneos system to extend functionality or simplify management.

A number of ready-made includes are available to help streamline common tasks and configurations within the Geneos system.

## Commands

| Command | Description |
| --- | --- |
| `geneos includes delete / remove` | Delete unlinked versions, optionally matching a version pattern |
| `geneos includes extract` | Extract embedded includes and configuration templates |
| `geneos includes install` | Install a versioned include archive and missing configuration templates |
| `geneos includes list` | List available and configured include files |
| `geneos includes update` | Update links to a versioned include set |

All commands can be limited to a configured host with the global `--host`/`-H` option. Use `geneos includes COMMAND --help` for command options and output details.

## Release Archives

The archive contains a `v`-prefixed semantic-version directory of managed include files and a separate, unversioned `config/` directory of configuration templates. Install it with `geneos includes install ARCHIVE.tar.gz`. Existing configuration templates are never replaced. Missing top-level include links are created; use `--update` to replace existing paths with links to the installed version.


## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos includes delete / remove`](geneos_includes_delete.md)	 | Delete unlinked include versions |
| [`geneos includes extract`](geneos_includes_extract.md)	 | Extract includes |
| [`geneos includes install`](geneos_includes_install.md)	 | Install includes from an archive |
| [`geneos includes list / ls`](geneos_includes_list.md)	 | List all available include files |
| [`geneos includes update`](geneos_includes_update.md)	 | Update includes |

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
