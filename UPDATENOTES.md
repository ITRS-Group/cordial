# Update Notes

This is a summary of breaking and other upgrade-sensitive changes in Cordial.
When updating across multiple versions, review every relevant section between
your installed version and the version you are installing. Versions without
selected update notes are omitted; their omission does not guarantee that an
update needs no attention. See the [changelog](CHANGELOG.md) for full details
and fixes, and [Highlights](HIGHLIGHTS.md) for new features.

## [v1.31.0](CHANGELOG.md#version-v1310) (in development)

* **Removed from releases:** The original `servicenow` integration,
  `pagerduty` and `san-config`, together with their configurations and
  documentation, are no longer shipped. Use the versions in v1.30.1 or build
  them from source. For ServiceNow, consider moving to `servicenow2` or
  `ims-gateway`; both remain in releases.
* **Release archive includes:** The main archive no longer contains
  `/etc/geneos/includes`. Use the separate versioned `geneos-includes` archive
  or the embedded files managed by `geneos includes` instead.
* **Includes commands:** Remove `--force`/`-F` from scripts using `geneos
  includes extract` or `update`. Re-extraction no longer requires it. Existing
  top-level files and links are only replaced with `extract --update`, `install
  --update` or `update`; existing configuration templates are not replaced.
* **Include list output:** Update consumers of JSON, pretty, CSV and Toolkit
  output for separate status and boolean fields instead of compact flags. CSV
  and Toolkit headings use camelCase; column and table headings use NormalCase.
  Gateway-local entries use `gatewayLocal` and `G`, not `local` and `L`. Paths
  shown for some files have also changed; gateway-local references retain their
  configured paths, except the main local setup is relative to the instance.
* **Package installation:** `-A` now means `--archive`, not `--all`; use
  `--all` explicitly. Local archive sources do not fall back to downloads, and
  a local `--archive` cannot be combined with `--download`. Explicit URLs are
  still fetched.
* **Site includes:** The default site name is `local`; configuration includes
  are installed under `gateway/<site>/<site>.*.xml`. Existing
  `gateway/includes/config` files are copied without overwriting existing
  files and retained in their original location. Existing user profiles and
  Gateway setups are not rewritten; check their paths before using the new
  layout.

## [v1.30.1](CHANGELOG.md#version-v1301)

v1.30.1 replaces v1.30.0.

* **Configuration keys:** Change nested-key references from `::` to `.`, for
  example `tls::verify` to `tls.verify`, in configurations and scripts.
  Special cases include domain-name and IP keys in remote-host configuration.
* **SAN and Floating Netprobe gateways:** The `gateways` configuration changed
  from an object to an array. Review and manually update affected instance
  configurations; when adding or removing gateways with `geneos set` or
  `unset`, supply the complete gateway list because the original list is
  deleted on change.
* **Templates:** Run `geneos init templates` after upgrading to use the new
  SAN and Gateway template features. Back up and compare any customised
  templates first.
* **Java truststores:** Special support for external command-line truststores
  was removed for `webserver` and `sso-agent`. Configure additional truststores
  through instance `options`; component-specific configuration is unaffected.
* **Automation and configuration:** `geneos list` separates the base symlink
  and package version in programmatic output. Configuration expansion can now
  repeat, which can affect configurations relying on single-pass expansion.
  `geneos ps --files` and `--network` are mutually exclusive.

## [v1.29.0](CHANGELOG.md#version-v1290)

This release was withdrawn; use v1.29.1 or later instead.
`gateway-reporter` was retired from builds in favour of the Gateway's built-in
reporting features.

## [v1.27.0](CHANGELOG.md#version-v1270)

* Back up customised templates and compare them with the updated Gateway and
  Netprobe templates. For existing installations, run `geneos init template`
  after upgrading. The changelog also recommends `geneos backup --shared --aes
  --tls` for a rollback option.
* `geneos tls info` columns, names and ordering changed; update output
  consumers. Remote process checks now only see processes owned by the
  configured user, which can affect detection by `ps`, `start` and `stop`.

## [v1.26.1](CHANGELOG.md#version-v1261)

v1.26.0 was withdrawn. The v1.26 series contains significant TLS changes:

* Existing TLS configurations continue to work, but using the new certificate
  and trust-chain features requires instance configuration changes. Read the
  [TLS changes](tools/geneos/TLS.md) and back up instance configurations
  before `geneos tls migrate`: migration is not reversible. TLS is enabled by
  default for new `init` and `deploy` installations unless disabled.
* Most `geneos` commands refuse to run as root without `--allow-root`; the
  global `allow-root=true` override is available but not recommended.
* To apply the corrected `listenip=none` default to existing SAN and Floating
  Netprobes, use `geneos set san listenip=none` and `geneos set floating
  listenip=none` for the affected instances.
* The `pkg/config` API has been refactored; external Go consumers must review
  removed and changed methods before updating their dependencies.

## [v1.24.1](CHANGELOG.md#version-v1241)

CSV reports no longer include the report title as their first line. Check
consumers that depend on the previous output layout.

## [v1.24.0](CHANGELOG.md#version-v1240)

GDNA's `csv` format is now standard CSV, separate from `toolkit` output.
Plugin lists in reports changed from `name(N)` to `name:N`, with spaces between
plugins. Update report consumers as needed.

## [v1.22.0](CHANGELOG.md#version-v1220)

The `cordial-geneos.xml` include was renamed to
`itrs.level0.geneos-cordial.xml`; check references and diagnostics consumers.
Previously deprecated command flags were removed. The reported local hostname
changed from `localhost` to the real hostname, although both remain aliases.

## [v1.21.0](CHANGELOG.md#version-v1210)

* `geneos ps` no longer has `--nolookup`/`-n`; `-n` now means `--network`.
  Port-list separators and table formatting also changed.
* Probe instances have a new `listenip` parameter. Remove any custom
  `-listenip` from an instance's `options` for the parameter to take effect.

## [v1.20.0](CHANGELOG.md#version-v1200---codename-leo)

Upgrade the GDNA binary before using the new Monitoring Coverage and Version
dashboards. The old dashboard works with the new binary.

## [v1.18.0](CHANGELOG.md#version-v1180)

* GDNA deprecated the original external filtering files; import their data
  using the new filter commands. Some YAML report options moved, including
  `enable-for-dataview` and `enable-for-xlsx`; update affected configurations.
  `gdna list` now uses subcommands such as `gdna list reports`.
* `geneos package install` defaults to component types already present; specify
  a type or `--all` to install others.

## [v1.17.1](CHANGELOG.md#version-v1171)

The Gateway `insecureport` parameter requires an updated
`gateway-instance.setup.xml.gotmpl`; update it manually or use `geneos init
templates`, taking care with customised templates.

## [v1.17.0](CHANGELOG.md#version-v1170)

`geneos package uninstall` uses `-F`, not `-f`, for `--force`, and `geneos
package update` no longer supports `--install`. Local archive file paths now
ignore `--version`; review installation scripts.

## [v1.15.1](CHANGELOG.md#v1151-changes)

Saved global configuration defaults may need to be removed manually to pick up
new defaults. `geneos set x+="y"` now inserts a space only when the appended
value begins with `-`. `geneos init san` and `init floating` were deprecated in
favour of `geneos deploy san` and `deploy floating`.

## [v1.14.0](CHANGELOG.md#v1140-changes)

The `geneos deploy` shorthand `-n` was repurposed for `--nosave`; review
scripts using it for `--name`.

## [v1.13.2](CHANGELOG.md#version-v1132)

v1.13.2 replaces v1.13.0 and v1.13.1. Review the
[full breaking changes](CHANGELOG.md#v1132-breaking-changes):

* TLS flags in `init` and `deploy` changed: `--tls`/`-T` enables TLS,
  `--signing-bundle`/`-C` supplies signing material, and
  `--instance-bundle`/`-c` supplies instance material. The old long forms
  `--secure` and `--makecerts` remain aliases; `--importkey` was deprecated.
  Check existing scripts, especially their `-C` and `-c` usage.
* The shorthand for `--override` changed to `-O` in commands supporting it.
  The behaviour and flags of `aes new`, `aes import` and `aes set` changed;
  review their help before reuse.
* `package uninstall` requires `--update`/`-U` to update instances using older
  releases; without it, referenced releases are protected.

## [v1.12.0](CHANGELOG.md#version-v1120)

Go consumers of `pkg/config` should review the changed `LookupTable()` options;
`LookupTables()` was deprecated.

## [v1.10.4](CHANGELOG.md#version-v1104)

`geneos aes import` no longer automatically updates instance key files; pass
`--update` when that behaviour is intended.

## [v1.10.0](CHANGELOG.md#version-v1100)

`pkg/config` no longer embeds Viper; it is now named `Viper`, and not all
previously promoted Viper methods are available. Review dependent Go code and
use the package's mutex-protected wrappers where available. `Sub()` now returns
an empty Config rather than `nil` for a missing key.

## [v1.8.0](CHANGELOG.md#version-v180)

* `geneos tls import` syntax and functionality changed; review existing scripts.
* External key files are enabled by default for new Gateways on GA5.14 and
  later. Existing Gateways are unaffected by the new default. Before changing
  key-file use for a Gateway that has already started and created a cache,
  follow the [Gateway key-file instructions](https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm#How_to_change_the_key_file_of_your_Gateway).

## [v1.7.0](CHANGELOG.md#version-v170)

`help` now shows the long description and usage; `--help`/`-h` shows the short
description and usage. `pkg/config` also had API changes; Go consumers should
review the changelog.

## [v1.6.4](CHANGELOG.md#version-v164)

Go code using `config.Plaintext` should review the change to
`*config.Plaintext`.

## [v1.6.1](CHANGELOG.md#version-v161)

Gateway configuration output structures in `pkg/geneos` were renamed with an
`Out` suffix down to `SamplerOut`; update dependent Go code.

## [v1.6.0](CHANGELOG.md#version-v160)

* Netprobe types moved under `netprobe/`. Existing installations should
  continue to work; `geneos migrate` can merge their directories and update
  configurations. Check customised SAN and Floating Netprobe templates for
  name clashes in the shared template directory.
* Some `pkg/config` `Load()` and `Save()` options were renamed without
  backwards compatibility; update dependent Go code.

## [v1.5.2](CHANGELOG.md#version-v152)

TLS root and signing certificate/key directories changed, and functionality
previously in `tls new` moved to `tls create`. Review scripts and paths.

## [v1.5.0](CHANGELOG.md#version-v150)

* Check SAN gateway configurations if connections fail; the gateway
  configuration format changed. Re-setting the gateways may help, but manual
  cleanup of `san.json` may still be necessary.
* SAN and Gateway variable storage changed. Old values are converted when the
  configuration is updated, but previously incorrect variable-name casing is
  not corrected automatically; review it.
* Running instances as other users or under `sudo` was deprecated. Do not rely
  on the old privilege-handling behaviour.
* `pkg/logger` was removed, and `pkg/config` and other public APIs changed.
  Update dependent Go code before upgrading.

## [v1.4.2](CHANGELOG.md#version-v142---20221221)

Remote hosts named by IP addresses are renamed to `A-B-C-D`; check scripts
that refer to those names.

## [v1.4.1](CHANGELOG.md#version-v141---20221219)

The `pkg/config` expansion functions gained options, changing their API. Go
consumers should update calls as necessary.

## [v1.2.1 release candidates](CHANGELOG.md#version-v121-rc3---20221007)

The release-candidate changes leading to v1.2.1 included removal of built-in
credential opaquing in output in favour of `${enc:...}`. Existing instance
configuration files are not automatically updated to use configuration
expansion; see the [rc1 notes](CHANGELOG.md#version-v121-rc1---20220928).
Support for `$var` expansion was removed in favour of `${var}` in
[v1.2.0-rc2](CHANGELOG.md#version-v120-rc2---20220926).

## [v1.2.0-rc1](CHANGELOG.md#version-v120-rc1---20220921)

The Geneos `api` and `api-streams` XML-RPC package APIs were substantially
changed. Other public package APIs also changed since v1.0.0; review dependent
Go code.
