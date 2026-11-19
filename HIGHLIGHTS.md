# Highlights

New features and notable capabilities, grouped by version, newest first. Versions without selected highlights are omitted. For upgrade-sensitive changes, see [UPDATENOTES](UPDATENOTES.md).

> [!NOTE]
> This is a short list of the most significant changes only. The [CHANGELOG](CHANGELOG.md) records notable features, changes and fixes; the [update notes](UPDATENOTES.md) cover breaking and upgrade-sensitive changes.

## [v1.31.0](CHANGELOG.md#version-v1310) (in development)

> [!NOTE]
> Built with Go 1.27.4.

* **Automation and reliability:** `geneos` reports failures consistently and returns a non-zero status when any requested item fails. Significant fixes protect existing files, instance directories and certificate stores after failed operations, restore broken installation workflows, and prevent credential exposure through logs, redirects or unintended configuration expansion.

* **Monitoring Health:** The supplied `health-agent` and `health-gateway` profiles replace Monitor-Of-The-Monitors. Monitoring includes, identifiers, GDNA reports and dashboards use `layer` rather than `level` names. Existing installations need migration; see the [update notes](UPDATENOTES.md#v1310-in-development).

* **Geneos management:** Profile options persist correctly, secrets are encoded before saving, and rebuilds show member actions. `set` and `unset` can rebuild and reload in one operation. TLS import and inspection support more certificate sources, package installation handles multiple archives, and `adb info`, `clean` and the new `edit` alias improve XPath and URL inspection and editing. Gateway XML updates better preserve existing content.

* **Gateway includes (preview):** Manage includes and templates independently of the `geneos` version, with source-prefixed versions, site-specific configuration and a separate archive. Different sources can coexist without one source's updates pruning another's links. New `geneos includes install`, `delete`/`remove` and `fix` commands install releases, prune unused versions and repair references. Existing setups using the old configuration naming can move to the site layout with `fix`. This remains a preview.

* **Incident integrations:** ServiceNow and ServiceDesk Plus handling now requires authentication and correlation, reports failures to clients, and avoids duplicate submissions after an incident is accepted. Request data stays literal, and configuration reloads refresh connection credentials.

* **Background processes:** `servicenow2 proxy`, `ims-gateway` and `gdna` support `stop`, `restart`, `reload` and `status`. Startup checks and locked PID files detect failures and prevent duplicate starts; reloads apply configuration without interrupting work in progress.

* **GDNA:** Failed fetches no longer leave partial source data, licence and coverage reports are corrected, and a failed run no longer stops scheduled processing. Reloaded source configuration takes effect, report errors prevent incomplete email delivery, and log settings are applied consistently.

* **Release contents:** Includes ship in their own versioned archive, alongside expanded release documentation. The original `servicenow`, `pagerduty` and `san-config` are no longer distributed; use v1.30.1 or build from source. For ServiceNow, use `servicenow2` or `ims-gateway`.

## [v1.30.1](CHANGELOG.md#version-v1301)

> [!NOTE]
> Built with Go 1.27.3.

v1.30.1 replaces v1.30.0. Monitor-Of-The-Monitors features are preview features.

* **Profiles and includes:** Deploy and manage groups of components using profiles, with embedded Geneos Best Practice includes for common setups.

* **Snapshots:** Export Gateway dataview snapshots in single- or multi-dataview Toolkit format.

* **Auditing:** Global and per-instance audit logs, with redaction of sensitive information from most user-facing output and audit logs.

* **Dashboard tools:** Inspect and clean Active Dashboard files with `geneos adb`.

* **Configuration:** Support multiple SAN managed entities and nested Gateway include groups using updated templates.

## [v1.29.0](CHANGELOG.md#version-v1290)

> [!NOTE]
> Built with Go 1.27.1.

This release was withdrawn; use v1.29.1 or later instead.

* `geneos` supports the Netprobe Router as the `router` component.

* Early Geneos Best Practice Gateway includes were added.

## [v1.28.1](CHANGELOG.md#version-v1281)

> [!NOTE]
> Built with Go 1.26.5.

`geneos init all` and `deploy` accept PFX/PKCS#12 signing bundles.

## [v1.28.0](CHANGELOG.md#version-v1280)

Configure CPU affinity for local Linux instances with `cpus`; `geneos ps --long` shows the running affinity.

## [v1.27.0](CHANGELOG.md#version-v1270)

* `geneos reset` provides a safer `clean --full` variant that matches no instances by default.

* `geneos tls info` gained verification and Toolkit output options; `ps` can identify managed Collection Agent child processes.

* Secure Gateway variables can be written to `instance.setup.xml` using the Gateway's AES key file.

* `libemail` added Microsoft Teams webhook support.

## [v1.26.1](CHANGELOG.md#version-v1261)

> [!NOTE]
> Built with Go 1.26.2.

* **TLS:** Improved certificate and trust-chain management, TLS by default for new installations, and `geneos tls migrate` for existing instances.

* **Incident management:** New `geneos incident` commands and IMS Gateway integration for ServiceNow and ServiceDesk Plus.

* **Restore:** `geneos init` can restore a `geneos backup` when creating an installation.

* **GDNA:** Support for fetching licence data from the protected License Daemon 7.8.0 and later endpoints, with additional configuration.

## [v1.24.4](CHANGELOG.md#version-v1244)

The ServiceNow v2 integration added an Action Group `exit` action and a `nomatch` expansion option.

## [v1.24.1](CHANGELOG.md#version-v1241)

GDNA can fetch and report on `licd` binary summary files; `dv2email` accepts a command-line subject override.

## [v1.24.0](CHANGELOG.md#version-v1240)

GDNA can run ad-hoc fetch and report cycles with command-line sources and produce ZIP archives of formatted reports.

## [v1.23.0](CHANGELOG.md#version-v1230)

> [!NOTE]
> Built with Go 1.25.3.

`geneos` accepts URLs for PEM inputs and custom HTTP headers for package downloads.

## [v1.22.1](CHANGELOG.md#version-v1221)

Initial native Windows support allows `geneos.exe` to install and manage Active Console instances after the environment is configured.

## [v1.22.0](CHANGELOG.md#version-v1220)

> [!NOTE]
> Built with Go 1.24.5.

`geneos backup` and `restore` can move or duplicate instances between servers, optionally including sensitive files and remapping instance names.

## [v1.21.0](CHANGELOG.md#version-v1210)

> [!NOTE]
> Built with Go 1.24.3.

* A new ServiceNow incident integration and a `geneos` self-monitoring include.

* Toolkit output for `geneos` list commands and `ps`, plus open-file and network connection views in `ps`.

## [v1.20.0](CHANGELOG.md#version-v1200---codename-leo)

GDNA introduced new Monitoring Coverage and Version dashboards; `geneos host` added SSH private-key support.

## [v1.19.0](CHANGELOG.md#version-v1190)

`geneos` can install and manage SSO Agent instances, including their keystore and truststore.

## [v1.18.0](CHANGELOG.md#version-v1180)

> [!NOTE]
> Built with Go 1.23.2.

GDNA added configurable plugin lists and include, exclude and grouping filters. Its reporting code became the reusable `pkg/reporter` package.

## [v1.17.1](CHANGELOG.md#version-v1171)

GDNA added a Version Summary dashboard and controls for dataview creation and conditional formatting. Gateway instances gained an `insecureport` parameter.

## [v1.17.0](CHANGELOG.md#version-v1170)

GDNA added Geneos 7 licence and Gateway data, reports and columns. `geneos` added a minimal Netprobe flavour and a `use-chain` parameter.

## [v1.15.3](CHANGELOG.md#version-v1153)

GDNA added an unused-Gateways report and corresponding coverage headlines.

## [v1.15.1](CHANGELOG.md#v1151-changes)

`geneos init demo` gained a minimal Netprobe option; GDNA added a `stop` command.

## [v1.15.0](CHANGELOG.md#v1150-changes)

Introduced GDNA (Geneos Dynamic Netprobe Analysis) for assessing monitoring coverage from licence data and plugin usage.

## [v1.14.0](CHANGELOG.md#v1140-changes)

> [!NOTE]
> Built with Go 1.22.4.

`geneos tls create` gained options for destination, certificate bundles and expiry; `gateway-reporter` added an optional XLSX password.

## [v1.13.2](CHANGELOG.md#version-v1132)

> [!NOTE]
> Built with Go 1.22.3.

`geneos tls export` helps share signing material between installations. Webserver instances gained automatic TLS keystore and trust-chain management.

## [v1.12.0](CHANGELOG.md#version-v1120)

* New `files2dv` tool scans file metadata and contents into Toolkit dataviews.

* `gateway-reporter` added a live CSV-directory output and optional Gateway include.

## [v1.11.0](CHANGELOG.md#version-v1110)

* `dv2email` gained XLSX and text-table attachments and dataview export.

* `gateway-reporter` introduced static Gateway configuration coverage reports.

* `geneos aes encode` gained app-key support for Gateway Hub and Obcerv Centralised Configs.

## [v1.10.0](CHANGELOG.md#version-v1100)

`geneos` added glob-style instance name matching for most commands, including multi-instance copy and move to a host.

## [v1.8.0](CHANGELOG.md#version-v180)

`geneos tls import` gained support for external instance certificates and chains. Start, restart and command operations gained one-off arguments and environment variables.

## [v1.7.0](CHANGELOG.md#version-v170)

`geneos` operations on instances run in parallel, and `ps` reports the version actually running rather than only the current base link.

## [v1.6.6](CHANGELOG.md#version-v166)

`geneos package install -D` downloads packages without unpacking them.

## [v1.6.4](CHANGELOG.md#version-v164)

`geneos` added hidden hosts and `show --validate` for JSON validation output.

## [v1.6.1](CHANGELOG.md#version-v161)

`geneos show` gained instance setup and merged Gateway configuration output; instances gained an `autostart` flag.

## [v1.6.0](CHANGELOG.md#version-v160)

`geneos deploy` combines initialisation and component creation for automation; `geneos migrate` helps consolidate Netprobe instance directories.

## [v1.5.2](CHANGELOG.md#version-v152)

`geneos tls create` can create certificates without depending on a Geneos home directory.

## [v1.5.0](CHANGELOG.md#version-v150)

* `geneos` introduced command subsystems, credential management, and `ca3` and Floating Netprobe components.

* New `dv2email` utility sends configurable dataview emails from Geneos.

## [v1.4.3](CHANGELOG.md#version-v143---20230412)

`geneos ps` gained open-file listing; `libemail` added initial Microsoft Teams notification support.

## [v1.4.1](CHANGELOG.md#version-v141---20221219)

`geneos` added per-host SSH password and private-key support.

## [v1.3.0](CHANGELOG.md#version-v130---20221025)

Added PagerDuty integration, merged ServiceNow integration, instance protection, legacy `*ctl` command emulation and remote-only `geneos` operations.

## [v1.2.1 release candidates](CHANGELOG.md#version-v121-rc3---20221007)

The changes leading to v1.2.1 included raw configuration output, enhanced AES commands and configuration-value expansion.

## [v1.2.0-rc1](CHANGELOG.md#version-v120-rc1---20220921)

Introduced `geneos aes` and `snapshot`, configuration expansion and encryption support, and early Geneos configuration and Gateway API packages.
