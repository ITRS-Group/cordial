# `geneos includes`

# Geneos Gateway Includes

Geneos Gateway includes package reusable monitoring definitions and editable configuration templates. They keep common functionality consistent across Gateways while allowing each Gateway to provide its own entities, variables, and overrides.

The `itrs.*.xml` files are functional includes. Treat them as read-only and upgrade them by installing a newer release. The `config.*.xml` files are local configuration templates; customize these or copy their settings into the Gateway setup. Installing files does not automatically add them to a Gateway's setup or enable monitoring.

## Files

Release archives are named `geneos-includes-v<version>.tar.gz` and have a single top-level directory named `geneos-includes-v<version>`. It contains the versioned functional includes under a `v`-prefixed semantic-version directory, an unversioned `config/` directory of templates, `scripts/install-includes.sh` for users installing without the `geneos` command, and a `docs/` directory with this README and the Markdown include documentation plus generated PDF versions. The `geneos includes install` command strips this wrapper and does not copy the script or documentation into the Gateway's live include directory.

| File | Purpose |
| --- | --- |
| `itrs.level0.foundation.xml` | Gateway information, license/severity views, and configuration validation tools. |
| `itrs.level1.infrastructure.xml` | Infrastructure monitoring for CPU, disk, hardware, and network. |
| `itrs.level2.basic-application.xml` | Baseline `Logs`, `Processes`, and Windows Services samplers for applications. |
| `itrs.level3.geneos.xml` | Geneos instance, component, and Gateway-load monitoring types, samplers, rules, and commands. |
| `config.level0.authentication.xml` | Authentication roles and policies plus example local users/groups; review and replace example identities before use. |
| `config.level0.foundation.xml` | Empty Gateway setup template for local foundation settings. |
| `config.level1.infrastructure.xml` | Gateway setup template for infrastructure configuration and overrides. |
| `config.level2.basic-application.xml` | Gateway setup template for application configuration. |
| `config.level3.geneos.xml` | Example managed-entity hierarchy, per-entity component variables, and sampling override environments for Level 3 Geneos. |
| `config.level4.analytics.xml` | Standalone Obcerv connection example; replace its endpoints and credentials and review TLS settings before use. There is no matching `itrs.level4.analytics.xml` functional include in this set. |

Choose a site name representing the company, customer, or client using the Geneos deployment, rather than a region or environment. For example, for Acme Corp, an installation with `geneos config set site=acme` has files organized like this:

```text
gateway/
├── includes/
│   ├── itrs.level0.foundation.xml -> v1.31.0/itrs.level0.foundation.xml
│   ├── itrs.level1.infrastructure.xml -> v1.31.0/itrs.level1.infrastructure.xml
│   ├── itrs.level2.basic-application.xml -> v1.31.0/itrs.level2.basic-application.xml
│   ├── itrs.level3.geneos.xml -> v1.31.0/itrs.level3.geneos.xml
│   └── v1.31.0/
│       ├── itrs.level0.foundation.xml
│       ├── itrs.level1.infrastructure.xml
│       ├── itrs.level2.basic-application.xml
│       └── itrs.level3.geneos.xml
└── acme/
    ├── acme.level0.authentication.xml
    ├── acme.level0.foundation.xml
    ├── acme.level1.infrastructure.xml
    ├── acme.level2.basic-application.xml
    ├── acme.level3.geneos.xml
    └── acme.level4.analytics.xml
```

The links let Gateway setups use a stable path while the target version changes. Configuration templates are regular files, copied only when absent and not overwritten by later installs. The archive itself always uses the `config` names listed above.

> [!NOTE]
> Please note the following when choosing and using a site name:
>
> - The site name is currently used for Gateway configuration include directories and filename prefixes, including paths in supplied profiles, but future releases may use it for other features.
> - Spaces become underscores in paths. The resulting name must be a safe single directory path component and cannot conflict with Gateway directories such as `config` or `includes`.
> - If no site name is set, the directory and prefix are `local`.
> - Only one site name is supported per installation.
> - Use the same site name on all Geneos servers for consistent paths.
> - Changing the site name later can have unforeseen effects: existing Gateway setups and user profile files are not rewritten. Review them and run `geneos includes install` to populate the new site directory.

On installation, files from the older `gateway/includes/config` directory are copied to the site directory before new templates are installed. Existing destination files are not replaced, and the old directory remains for previous Gateway setups.

## Installation

### Install with the `geneos` command

Install the includes embedded in the current `geneos` executable, or install a published archive:

```shell
geneos config set site=acme
geneos includes install
geneos includes install /path/to/geneos-includes-v1.31.0.tar.gz
```

Add `--update` to replace existing top-level include paths with links to the installed version.

### Install with shell tools

Use the external installer included in the release archive - `scripts/install-includes.sh`. Extract the archive, change into its versioned top-level directory, and run the script with the Geneos installation home as its only positional argument. The script uses the current directory as its source; set `INCLUDES_SOURCE` to use another extracted source directory.

```shell
tar -xzf /path/to/geneos-includes-v1.31.0.tar.gz
cd geneos-includes-v1.31.0
./scripts/install-includes.sh /path/to/geneos
```

The script installs into a fresh version directory, keeps existing config templates, and creates only missing top-level links. It uses `gateway/local/local.*.xml` and does not read the `geneos` site setting; use `geneos includes install` for a site name such as `acme`. It is intended for initial installation and does not replace the `geneos includes install` command's archive validation, backups, or Gateway-reference checks. If the version is already present, or you need to update live links, use `geneos includes install --update` or inspect and back up the affected files first.

## How-To

Installing include files only makes them available on disk. Use the Gateway Setup Editor (GSE) to add the include files to each Gateway setup and attach the provided types to the appropriate managed entities. Reference the stable top-level link for functional includes, for example `../../includes/itrs.level1.infrastructure.xml`, and the site template, for example `../../acme/acme.level1.infrastructure.xml`; adjust relative paths for the Gateway setup location and your site name. Do not reference the version directory directly. Include priorities must be unique. For a functional include and its local config, give the config include a numerically lower priority so its settings take precedence. Review each template before enabling it.

### Manage with the `geneos` command

| Command                                    | Task                                                        |
| ------------------------------------------ | ----------------------------------------------------------- |
| `geneos includes list`                     | List configured/available includes                          |
| `geneos includes list --all`               | Include version directories in the listing                  |
| `geneos includes list --gateway`           | Show include use by Gateway                                 |
| `geneos includes update --version v1.31.0` | Point top-level links at a version                          |
| `geneos includes fix --dry-run`            | Preview repairs for broken include references               |
| `geneos includes fix`                      | Repair broken references with unique same-name replacements |
| `geneos includes delete 'v1.30.*'`         | Delete unused version directories and their links           |

Both the `geneos includes update` and `geneos includes delete` commands check Gateway setups before removing obsolete links or versions. The `geneos includes install --update` command also updates links while installing. Use `geneos includes COMMAND --help` to see available options; the global `--host`/`-H` option selects the destination host.

### Manage with shell tools

List installed include files and broken top-level links:

```shell
INCLUDES_DIR=/path/to/geneos/gateway/includes
find "$INCLUDES_DIR" -maxdepth 2 \( -type f -o -type l \) -name '*.xml' -print | sort
find "$INCLUDES_DIR" -maxdepth 1 -type l -xtype l -print
```

To switch links to an already installed version, first inspect the Gateway setup references and back up any regular files at the link paths. Then repoint links for files present in the selected version:

```shell
VERSION=v1.31.0
INCLUDES_DIR=/path/to/geneos/gateway/includes

for source in "$INCLUDES_DIR/$VERSION"/itrs.*.xml; do
    [ -f "$source" ] || continue
    name=${source##*/}
    link="$INCLUDES_DIR/$name"
    if [ -e "$link" ] && [ ! -L "$link" ]; then
        cp -p "$link" "$link.$(date +%Y%m%d%H%M%S).bak"
    fi
    ln -sfn "$VERSION/$name" "$link"
done
```

This updates matching links only. It does not detect obsolete references or remove old links. Before removing an old version, inspect its links and every Gateway setup that may refer to them; `geneos includes delete` performs broader reference checks.

```shell
OLD_VERSION=v1.30.0
INCLUDES_DIR=/path/to/geneos/gateway/includes

# Preview top-level links with a direct relative target under this version.
# Inspect absolute links and Gateway setup references separately.
find "$INCLUDES_DIR" -maxdepth 1 -type l -lname "$OLD_VERSION/*" -print

# Only after confirming no Gateway setup uses this version:
find "$INCLUDES_DIR" -maxdepth 1 -type l -lname "$OLD_VERSION/*" -exec rm -f -- {} +
rm -rf -- "$INCLUDES_DIR/$OLD_VERSION"
```

For a broken link, inspect its target with `readlink` and confirm there is exactly one intended replacement with the same basename before recreating it with `ln -s`. The `geneos includes fix` command performs this matching and Gateway-reference work automatically.


## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos includes delete / remove`](geneos_includes_delete.md)	 | Delete unused include versions and links |
| [`geneos includes fix`](geneos_includes_fix.md)	 | Repair broken gateway include references |
| [`geneos includes install / extract`](geneos_includes_install.md)	 | Install embedded includes or includes from an archive |
| [`geneos includes list / ls`](geneos_includes_list.md)	 | List all available include files |
| [`geneos includes update`](geneos_includes_update.md)	 | Update includes |

### Options

```text
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string           profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
