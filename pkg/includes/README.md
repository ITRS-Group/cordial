# Geneos Gateway Includes

Geneos Gateway includes package reusable monitoring definitions and editable configuration templates. They keep common functionality consistent across Gateways while allowing each Gateway to provide its own entities, variables, and overrides.

The `itrs.*.xml` files are functional includes. Treat them as read-only and upgrade them by installing a newer release. The `config.*.xml` files are local configuration templates; customize these or copy their settings into the Gateway setup. Installing files does not automatically add them to a Gateway's setup or enable monitoring.

## Monitoring Layers

The Geneos Best Practice layers describe the scope and depth of monitoring, from the monitoring platform itself through infrastructure and applications to services and business activity. Layer numbers are not file versions or include priorities.

| Layer | Meaning |
| --- | --- |
| **0 - Foundation** | Shared Gateway configuration and monitoring-platform services, such as authentication, severity definitions and configuration validation. |
| **1 - Infrastructure** | Server health: CPU, memory, disk, hardware and network monitoring. The supplied include calls this `Server Health`. |
| **2 - Application Baseline** | Common monitoring building blocks for applications, including processes, logs and Windows services. |
| **3 - Applications** | Application-specific monitoring that builds on the baseline where appropriate, such as Geneos instance, component and Gateway-load monitoring. |
| **4 - End-to-End** | Monitoring complete service paths across application and infrastructure boundaries. |
| **5 - Business Activity** | Monitoring business transactions, activity and outcomes rather than only the underlying technology. |

The supplied functional includes currently cover Layers 0 to 3.

## Files

Release archives are named `geneos-includes-v<version>.tar.gz` and have a single top-level directory named `geneos-includes-v<version>`. It contains functional includes under `itrs-v<version>/`, an unversioned `config/` directory of templates, `scripts/install-includes.sh` for users installing without the `geneos` command, and a `docs/` directory with this README and the Markdown include documentation plus generated PDF versions. Installation strips the wrapper but preserves the `itrs-` prefix; the script and documentation are not copied into the Gateway's live include directory.

The prefix is the include **source**, independently of the deployment's site. User-created archives can use another source, for example `acme-ops-v1.2.3/` containing `acme-ops.*.xml`. Source names begin with a letter or digit and use letters, digits, dots, underscores or hyphens; `config` is reserved for templates. The suffix is a `v`-prefixed semantic version; prerelease and build metadata are supported, and the archive wrapper's version must match this suffix. Legacy unprefixed semantic-version directories remain supported as the `itrs` source. Listings show the full directory name in version columns.

| File | Purpose |
| --- | --- |
| `itrs.layer0.foundation.xml` | Gateway information, license/severity views, and configuration validation tools. |
| `itrs.layer1.server-health.xml` | Server health monitoring for CPU, disk, hardware, and network interfaces. |
| `itrs.layer2.programs-and-logs.xml` | Shared `Logs`, `Processes`, and Windows Services samplers for the Application Baseline. |
| `itrs.layer3.geneos.xml` | Geneos instance, component, and Gateway-load monitoring types, samplers, rules, and commands. |
| `config.layer0.authentication.xml` | Authentication roles and policies plus example local users/groups; review and replace example identities before use. |
| `config.layer0.foundation.xml` | Empty Gateway setup template for local foundation settings. |
| `config.layer1.server-health.xml` | Gateway setup template for server health configuration and overrides. |
| `config.layer2.programs-and-logs.xml` | Gateway setup template for Layer 2 Application Baseline configuration. |
| `config.layer3.geneos.xml` | Example managed-entity hierarchy, per-entity component variables, and sampling override environments for Layer 3 Geneos. |

Holiday calendars are a separate optional corpus under the `holidays` source,
distributed as `geneos-holidays-include-v<version>.tar.gz`. They are not embedded
in the `geneos` executable or installed by `geneos init` or a no-argument
`geneos includes install`. Install the archive explicitly:

```shell
geneos include install /path/to/geneos-holidays-include-v1.31.0.tar.gz
```

Its documentation is in the archive's `docs/README.md` and regional pages
(source tree: `pkg/holidays/`). Installing it makes calendars available but
does not add them to Gateway setups or enable holiday scheduling.

Choose a site name representing the company, customer, or client using the Geneos deployment, rather than a region or environment. For example, an installation with `geneos config set site=acme`, ITRS includes and an additional `acme` source has files organized like this:

```text
gateway/
├── includes/
│   ├── itrs.layer0.foundation.xml -> itrs-v1.31.0/itrs.layer0.foundation.xml
│   ├── acme.trading.xml -> acme-v1.2.3/acme.trading.xml
│   ├── ...
│   ├── itrs-v1.31.0/
│   │   ├── itrs.layer0.foundation.xml
│   │   └── ...
│   └── acme-v1.2.3/
│       ├── acme.trading.xml
│       └── ...
└── acme/
    ├── acme-config.layer0.authentication.xml
    ├── acme-config.trading.xml
    └── ...
```

The links let Gateway setups use a stable path while the target version changes. Configuration templates are regular files named `<site>-config.*.xml`, copied only when absent and not overwritten by later installs. Keeping `-config` in the filename distinguishes a site override from a functional include even when source and site have the same name. The archive itself always uses the `config` names listed above. Templates from different sources share the site directory, so choose distinct template basenames unless they are deliberately shared.

> [!NOTE]
> Please note the following when choosing and using a site name:
>
> - The site name is currently used for Gateway configuration include directories and filename prefixes, including paths in supplied profiles, but future releases may use it for other features.
>
> - Spaces become underscores in paths. The resulting name must be a safe single directory path component and cannot conflict with Gateway directories such as `config` or `includes`.
>
> - If no site name is set, the directory and prefix are `local`.
>
> - Only one site name is supported per installation.
>
> - Use the same site name on all Geneos servers for consistent paths.
>
> - Changing the site name later can have unforeseen effects: existing Gateway setups and user profile files are not rewritten. Review them and run `geneos includes install` to populate the new site directory.

On installation, referenced files from the older `gateway/includes/config` directory and the previous `<site>.*.xml` site naming are copied to `<site>-config.*.xml` before new templates are installed. Existing destination files are not replaced, and originals remain for previous Gateway setups. Run `geneos includes fix` to migrate references where the old and new contents match. The standalone installer cannot check references and copies all legacy config files.

## Installation

### Install with the `geneos` command

Install the includes embedded in the current `geneos` executable, or install a published archive:

```shell
geneos config set site=acme
geneos includes install
geneos includes install /path/to/geneos-includes-v1.31.0.tar.gz
```

Add `--update` to replace existing top-level include paths with links to the installed version. Obsolete-link cleanup is restricted to that source, and updates refuse to replace a link owned by another source.

`includes update`, `list` and `delete` accept `--prefix`, defaulting to `itrs`.
For example, `geneos include update --prefix holidays` selects the installed
holiday version matching the running release; add `--version v1.31.0` to
choose another version. Full source-version names still work with `update
--version` without a prefix flag. List and delete accept `--prefix '*'` to
include all sources; use it for the previous complete inventory or cross-source
cleanup. Install/extract infer the source from an archive unless an explicit
`--prefix` is supplied to require a matching source. No-argument installation
supports only the embedded ITRS source.

> [!IMPORTANT]
> Include and site filenames have changed from `levelN` to `layerN`, and Gateway types, groups, environments and XPath targets now use `Layer N` names. The former `itrs.level1.infrastructure.xml` and `itrs.level2.application-core.xml` are now `itrs.layer1.server-health.xml` and `itrs.layer2.programs-and-logs.xml`. Back up your configuration and install without `--update`; copy or merge site overrides into the new site files without overwriting existing settings. Update Gateway setups, SANs, customised profiles and any other references before running `geneos includes update`. Existing configurations are not renamed automatically.

### Install with shell tools

Use the external installer included in the release archive - `scripts/install-includes.sh`. Extract the archive, change into its versioned top-level directory, and run the script with the Geneos installation home as its only positional argument. The script uses the current directory as its source; set `INCLUDES_SOURCE` to use another extracted source directory.

```shell
tar -xzf /path/to/geneos-includes-v1.31.0.tar.gz
cd geneos-includes-v1.31.0
./scripts/install-includes.sh /path/to/geneos
```

The script installs into a fresh version directory, keeps existing config templates, and creates only missing top-level links. It uses `gateway/local/local-config.*.xml` and does not read the `geneos` site setting; use `geneos includes install` for a site name such as `acme`. It is intended for initial installation and does not replace the `geneos includes install` command's archive validation, backups, or Gateway-reference checks. If the version is already present, or you need to update live links, use `geneos includes install --update` or inspect and back up the affected files first.

The holiday archive bundles the same source-aware shell installer. Archives
without a `config/` directory, including holidays, do not create or migrate site
configuration. Each archive installs only its own source and preserves existing
top-level files and links.

## How-To

Installing include files only makes them available on disk. Use the Gateway Setup Editor (GSE) to add the include files to each Gateway setup and attach the provided types to the appropriate managed entities. Reference the stable top-level link for functional includes, for example `../../includes/itrs.layer1.server-health.xml`, and the site template, for example `../../acme/acme-config.layer1.server-health.xml`; adjust relative paths for the Gateway setup location and your site name. Do not reference the version directory directly. Include priorities must be unique. For a functional include and its local config, give the config include a numerically lower priority so its settings take precedence. Review each template before enabling it.

### Manage with the `geneos` command

| Command                                    | Task                                                        |
| ------------------------------------------ | ----------------------------------------------------------- |
| `geneos includes list`                     | List configured/available includes                          |
| `geneos includes list --all`               | Include version directories in the listing                  |
| `geneos includes list --gateway`           | Show include use by Gateway                                 |
| `geneos includes update --version itrs-v1.31.0` | Point top-level links at a version                          |
| `geneos includes fix --dry-run`            | Preview repairs for broken include references               |
| `geneos includes fix`                      | Repair broken references with unique same-name replacements |
| `geneos includes delete 'itrs-v1.30.*'`         | Delete unused version directories and their links           |

Both the `geneos includes update` and `geneos includes delete` commands check Gateway setups before removing obsolete links or versions. The `geneos includes install --update` command also updates links while installing. Use `geneos includes COMMAND --help` to see available options; the global `--host`/`-H` option selects the destination host.

### Auditing

The `install`/`extract`, `update`, `delete`/`remove` and `fix` commands follow the standard `audit.commands` and `audit.actions` settings, enabled by default. `list` is not audited.

File action records include the host, full file path, result and relevant source, link target or backup path. Extractions, backups, legacy copies, link changes and individual deletions are recorded without instance details in the global audit log. Unchanged or preserved files are reported on the console but are not recorded as filesystem changes.

Gateway setup and saved-configuration repairs by `fix` instead include Gateway instance details and a description of each old-to-new include location, with the setup backup path where available. These use the standard instance audit logging, including the global log and a local instance log where applicable. `fix --dry-run` audits the command invocation but does not record file changes.

### Manage with shell tools

List installed include files and broken top-level links:

```shell
INCLUDES_DIR=/path/to/geneos/gateway/includes
find "$INCLUDES_DIR" -maxdepth 2 \( -type f -o -type l \) -name '*.xml' -print | sort
find "$INCLUDES_DIR" -maxdepth 1 -type l -xtype l -print
```

To switch links to an already installed version, first inspect the Gateway setup references and back up any regular files at the link paths. Then repoint links for files present in the selected version:

```shell
VERSION=itrs-v1.31.0
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
OLD_VERSION=itrs-v1.30.0
INCLUDES_DIR=/path/to/geneos/gateway/includes

# Preview top-level links with a direct relative target under this version.
# Inspect absolute links and Gateway setup references separately.
find "$INCLUDES_DIR" -maxdepth 1 -type l -lname "$OLD_VERSION/*" -print

# Only after confirming no Gateway setup uses this version:
find "$INCLUDES_DIR" -maxdepth 1 -type l -lname "$OLD_VERSION/*" -exec rm -f -- {} +
rm -rf -- "$INCLUDES_DIR/$OLD_VERSION"
```

For a broken link, inspect its target with `readlink` and confirm there is exactly one intended replacement with the same basename before recreating it with `ln -s`. The `geneos includes fix` command performs this matching and Gateway-reference work automatically.
