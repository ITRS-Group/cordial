# Update Notes

This records breaking and other upgrade-sensitive changes in Cordial. When updating across multiple versions, review every relevant section between your installed version and the version you are installing. Versions without selected update notes are omitted; their omission does not guarantee that an update needs no attention. See the [CHANGELOG](CHANGELOG.md) for notable changes and fixes, and [HIGHLIGHTS](HIGHLIGHTS.md) for release highlights.

## [v1.31.0](CHANGELOG.md#version-v1310) (in development)

> [!NOTE]
> Built with Go 1.27.4.

### `geneos` automation and configuration

* **Breaking change - exit status:** Commands now exit with a non-zero status if any requested instance, host, file or other item fails, rather than reporting failures and exiting successfully. Other items are still processed where supported. Explicit names that do not exist are failures; unmatched patterns or component types with no instances are not. Commands that validate all names first still fail before acting. Already-running starts, stopped-instance stops/restarts/reloads, unsupported actions and disabled instances skipped by a start-all are not failures. Review scripts and Geneos actions that relied on a zero exit status after an error; this also applies to legacy `ctl` commands.

* **Stricter validation:** Unknown hosts are rejected, never treated as all hosts. `add` and `deploy` reject existing instances, although `deploy` and `init` still reapply existing profiles. Undefined profiles and unsupported multi-component profile port overrides are errors. Failures previously only logged by package, TLS, restore, host, AES and configuration commands now fail the command.

* **Configuration files and output:** `geneos config set`, `unset` and `show` use the file selected with `--config`/`-G`. Empty JSON results are `[]`, not `null`, and `tls list` reports instance certificate `remaining` as a duration. Check output consumers.

* **Initialization and trust:** `geneos init`, including with `--force`, keeps existing user-wide root and signing certificates and creates only missing ones. Use `geneos tls init --overwrite` to replace them deliberately. Root certificates that are expired or not valid root CAs are rejected by commands that read them.

* **Profile updates:** `set profile` and `unset profile` now change saved options and reapply the profile, rather than modifying members directly. Update the profile YAML or saved options for lasting member changes. Reapplying a profile also removes environment entries no longer defined by it, for all component types.

* **Profile TLS settings:** Reapplying a profile follows its saved `insecure` option, including for existing members. A saved insecure profile removes member TLS parameters even if TLS was enabled by hand; certificate files are kept, and a Web Dashboard keystore is renamed with a `.disabled` suffix. A secure profile requires a usable signing certificate. Profiles without a saved `insecure` option are unaffected. Review saved options before a deploy, rebuild or set.

* **Ports and TLS defaults:** Custom ports are preserved. Conventional Gateway and Web Dashboard TLS ports change only when the matching port is available; an occupied target port is no longer replaced by an arbitrary free port. SAN and floating Netprobe gateway ports change only for the conventional 7038/7039 pair. TLS-enabled instances now always receive the instance or global minimum TLS version, defaulting to `1.2`.

* **Rebuild and reload:** Profile reloads now reach members, and forced profile rebuilds reload even when generated files are unchanged. Rebuild-driven reloads skip `config.rebuild=never`. The new force action on `set` and `unset` implies rebuild and reload, but does not override `never`. Review automation that previously depended on these operations having no effect.

* **Archive and download handling:** Download timeouts now detect connection delays and stalled transfers, not total transfer duration. Non-file ZIP inputs are staged locally and limited to 8 GiB by default; change `download.max-archive-size` or `ITRS_DOWNLOAD_MAX_ARCHIVE_SIZE` if necessary. ZIP symlinks are skipped, or rejected if they escape the destination. `restore` requires a recognizable archive extension or an explicit decompression type for STDIN, refuses terminal input, and `--list` no longer writes or rebuilds anything.

* **Audit retention:** Previously ignored rotation settings now take effect, including the default 10 MiB size, 30 days, 5 backups and gzip compression. Set instance or global `audit` settings if older logs must be retained longer.

* **AES options:** `aes set` reads a keyfile from STDIN only without `--keyfile` or `--crc`. `--crc` cannot be combined with `--keyfile` or `--shared`. Check scripts relying on the previously ignored CRC option.

* **Dashboard cleaning reports:** `geneos adb clean` (also `edit`) adds `--verbose`/`-v` for individual before-and-after changes and `--log FILE`/`-l FILE` to append reports and diagnostics to a file rather than the console. `--output`/`-o` still selects the cleaned-dashboard directory. Logs can contain dashboard names and URLs.

* **TLS inventory output:** `geneos tls info` now names its first column `Source` in all output formats, replacing `FileAndIndex` or `PathAndIndex`. With `--leaf-only`, numeric certificate indexes are omitted, but keystore aliases are retained. Update CSV, Toolkit and other output consumers.

### `geneos` sites, includes and packages

* **Site names:** You can set a site name with `geneos config set site=acme` or `--site`/`-s` when using `geneos init` or local `geneos deploy`. The default site name is `local`. You should use the same site name for all Geneos deployments.

  At this time the only function of the site name is to determine the location where configuration includes are installed, which is under `gateway/<site>/<site>-config.*.xml`, instead of the previous `gateway/includes/config/config.*.xml` or `gateway/<site>/<site>.*.xml` locations. The `-config` infix avoids duplicate basenames when a functional include source has the same name as the site. This will likely expand in the future to offer further customization and control over site-specific configurations.

  Existing files in `gateway/includes/config` and files using the previous site naming that are used by a Gateway, as shown by `geneos includes list --gateway`, are copied and renamed appropriately without overwriting existing files and retained in their original location. The standalone installer cannot check Gateway setups and copies all the files for its `local` site. Existing user profiles and Gateway setups are not rewritten; check their paths before using the new layout, or use `geneos includes fix` to update references to files that are unchanged.

* **Include sources and versions:** New ITRS archive contents and embedded installs use `gateway/includes/itrs-v<version>` rather than `gateway/includes/v<version>`. The archive filename and `geneos-includes-v<version>` wrapper are unchanged. The prefix is the **source**; user-created archives can use another source, with matching `<source>.*.xml` files. Pass the full source-prefixed directory name to `includes update --version` and include it in `includes delete` patterns. Version columns retain this full name. Legacy unprefixed directories remain usable and count as the ITRS source; existing directories are not renamed. Updates clean obsolete links only for the selected source and reject replacement links owned by another source. Templates from all sources share the site directory and never overwrite existing destinations; choose distinct template basenames where separate configuration is required.

* **Independent include versions:** Standard includes are versioned by `pkg/includes/VERSION`, initially `v1.0.0`, independently of Cordial. Embedded installation and default include updates target `itrs-v1.0.0`, not the Cordial release version. Holiday snapshots use `pkg/holidays/VERSION` in `vYYYY.MM.DD` format, initially `v2026.11.20`, recording when the source data was fetched. Month/day are zero-padded and dates are validated; rebuilding an archive does not advance its snapshot date. Release archives and checksums use each corpus's own version.

* **Optional holiday calendars:** A new optional holiday corpus is published as `geneos-holiday-includes-vYYYY.MM.DD.tar.gz`. Install with `geneos include install ARCHIVE`; initialization and no-argument include installs provide only standard ITRS includes. The holiday archive uses the installer-compatible `geneos-includes-vYYYY.MM.DD` wrapper with `holidays-vYYYY.MM.DD/holidays.*.xml` contents. Add the required `holidays.layer0.<region>.xml` links to Gateway setups to make the named calendars available. For subsequent holiday link upgrades, install its archive with `--update` or use `includes update --prefix holidays`, optionally with `--version vYYYY.MM.DD`. Source assets live in `pkg/includes` and `pkg/holidays`.

* **Include prefix selection:** `includes update`, `list` and `delete` now accept `--prefix`, defaulting to `itrs`. Without `--version`, ITRS updates select the embedded standard corpus version; other prefixes, including holidays, select their newest installed version. Use `--prefix holidays --version vYYYY.MM.DD` for a specific snapshot. Full source-version update names remain supported; an explicit conflicting prefix is rejected. **List and delete now default to ITRS only**: use `--prefix '*'` for the previous complete inventory or cross-source cleanup, including when supplying delete patterns. Install/extract still infer an archive's source unless `--prefix` is explicitly supplied to require a match; non-ITRS sources require an archive.

* **Holiday shell installation:** The holiday archive now includes `scripts/install-includes.sh`, shared with the standard includes. Run it from the extracted archive wrapper with the Geneos home as its argument. It installs a fresh source-version directory and only missing links, leaving existing links, other sources and site files alone. Existing versions are rejected; use the CLI for managed updates. Archives without a `config/` directory no longer trigger shell-installer site configuration migration.

* **Low-confidence holiday calendars:** Active times containing any Low-confidence source data, including mixed-confidence calendars, are distributed disabled. Copy a calendar into your own writable Gateway setup or site include, verify its suitability, and edit dates manually if needed before enabling the reviewed local copy. Leave the distributed definition disabled.

* **Holiday collection process:** The holiday archive includes `docs/AGENT.md` and its PDF version, explaining the monthly source-fetching, date-conversion, confidence-rating and validation workflow for maintainers, LLM agents and users.

* **Minor breaking change — package installation:** `-A` now means `--archive`, not `--all`; use `--all` explicitly. Local archive sources do not fall back to downloads, and a local `--archive` cannot be combined with `--download`. Explicit URLs are still fetched.

* **Minor breaking change — includes commands:** Remove `--force`/`-F` from scripts using `geneos includes extract` or `update`; re-extraction no longer requires it. Existing top-level files and links are only replaced with `extract --update`, `install --update` or `update`. Existing configuration templates are never replaced.

* **Minor breaking change — include list output:** Update consumers of JSON, pretty, CSV and Toolkit output for separate status and boolean fields instead of compact flags. CSV and Toolkit headings use camelCase; column and table headings use NormalCase. Long version views name the second, previously duplicate `Version` column `TargetVersion` (`targetVersion` in machine output), so JSON no longer loses the directory's version label. Gateway-local entries use `gatewayLocal` and `G`, not `local` and `L`. Paths shown for some files have also changed; gateway-local references retain their configured paths, except the main local setup is relative to the instance.

* **Referenced include versions:** `update`, `delete` and `install --update` check Gateway references before removing obsolete links. A referenced link without a same-basename replacement blocks the update; migrate Gateway references and site overrides before removing renamed includes.

* **Shared include backups:** `geneos backup --shared` now preserves shared-directory symlinks without following them. Absolute targets within the installation are saved as portable relative links, and internal dangling links are retained. External targets cause an error. `restore --shared` and `init --restore` restore these links, keep existing destinations and reject writes through symlinked directories. Recreate older backups if they need include links that were previously omitted.

* **Initialization of existing includes:** Component initialization fills missing bundled files and links without replacing existing include files, site configuration files or links to other versions. During `init --restore`, archive files take precedence, including with `--force`; initialization does not upgrade restored includes to the bundled version. Use `geneos includes update` explicitly when an upgrade is intended.

### Monitoring Health and layer renaming

* **Monitoring Health:** The supplied `motm` and `motm-gateway` profiles are now `health-agent` and `health-gateway`. Instance names and the `CATEGORY=MOTM` and `COMPONENT=MOTM-SAN` attributes also change. Existing profiles, saved options and Gateway references are not migrated automatically; see the [profile migration guidance](tools/geneos/internal/component/profile/profile.md#use-a-supplied-profile).

* **Health profile attributes:** Supplied health profiles now use `SUBCATEGORY=Health Monitoring` or `SUBCATEGORY=Components`, with `COMPONENT` identifying the monitored component instead of `SUBCOMPONENT`. Review dashboards, filters and grouping based on the previous attribute hierarchy.

* **Monitoring layers:** Include and site filenames, Gateway types and other identifiers now use `layer` rather than `level`. Existing Gateway setups, SANs, profiles and site overrides must be migrated; see the [include upgrade guidance](pkg/includes/README.md#install-with-the-geneos-command).

* **GDNA layers:** Plugin configuration keys, report identifiers, Dataviews and dashboard bindings use layer names. Update customised configuration and dashboards as described in the [GDNA upgrade guidance](gdna/README.md#upgrading-from-level-based-names).

### `gdna` options

* **Minor breaking change — GDNA configuration file:** `gdna` now exits with an error if the file given with `--config`/`-f` cannot be read, instead of silently using the defaults. Check scripts and container commands that pass a file that may not exist.

* **GDNA log destination:** `gdna start` writes its log to the file set by `--logfile`/`-l` or `gdna.log.filename` (default `./gdna.log`), not to the console, and the `stop`, `restart`, `reload` and `status` commands no longer write to that file at all. Anything that relied on `gdna` logging to STDOUT/STDERR without options, such as a container, must now pass `-l -`. The supplied GDNA Docker image already does this in its entrypoint, so `docker logs` shows the logs; if you have your own image or entrypoint, or override the entrypoint in your compose file, add `-l -` to the `gdna start` command.

* **GDNA Netprobe options:** The `--hostname`/`-H` option of `gdna start` and `gdna report`, and the `--tls`/`-T` option of `gdna report`, were ignored in favour of the configuration and now take effect. Check scripts that pass them with values that differ from your configuration.

* **Source names:** HTTP/HTTPS licence source names now include an explicit URL port, for example `http:server:7041`. Existing database source names and `source` include, exclude or group patterns using the old name will not match; update them. URLs without a port are unchanged.

* **Retention and timeouts:** The documented `gdna.log.max-age` now takes effect, defaulting to 14 days rather than the previous hard-coded 30. Set it to `30` to retain the old behavior; an explicit legacy `gdna.log.stale-after` remains supported. SMTP uses the documented `email.timeout` (default `10s`), with legacy `_smtp_timeout` still honored when set.

* **Reports and filters:** An unreadable filters file, other than a missing file, or failed filter/group/allocation loading now fails the run. Report errors also fail the run, and an email is withheld if any report in it fails. Split reports require `split-values-query`. Reports run in name order, so a single-report output uses the first match by name; `${report-name}` and `${report-group}` now have their documented meanings. XLSX email attachments now honor `email.scramble`, which defaults to enabled.

### Running in the background

* **`--daemon` and PID files:** For `servicenow2 proxy`, `ims-gateway` and `gdna start`, `--daemon` now waits for the server to be listening, or `gdna` to be running its schedules, exits with a non-zero code if it fails to start, and holds a PID file, in the user's cache directory unless set with `--pidfile` or `server.pidfile` (`gdna.pidfile` for `gdna`), which prevents a second instance starting with the same file. If you run more than one on a server, give each its own PID file. The `stop`, `restart`, `reload` and `status` commands find the process using this file, so give them the same configuration file or `--pidfile` option that the process uses. `reload` and the `SIGUSR1` signal are not available on Windows; use `restart`.

* **GDNA background process:** `gdna` previously found its background process by name, so a process started by an earlier version has no PID file and `gdna stop` will not find it: stop it, for example with `kill`, before starting the new version. A foreground `gdna start` only holds a PID file if one is configured, and `--once` never does. `gdna restart`, which previously did nothing, now works, and `gdna stop` and CTRL+C cancel a fetch or report that is running.

### Incident integrations (`servicenow2`, `ims-gateway` and `geneos incident`)

* **Authentication and listening address:** The `servicenow2` proxy and `ims-gateway` refuse to start without `server.authentication.token`; the proxy also requires `server.listen`. The gateway previously accepted requests with no `Authorization` header if the token was empty.

* **Example gateway configuration:** If you copied the response messages from an earlier `ims-gateway.example.yaml`, replace `${_number}`, `${_error}` and similar with the `__` prefixed names, such as `${__number}`, which are the only ones the gateway supplies; the others expand to nothing.

* **Responses, logs and timeout:** The `servicenow2` proxy and IMS Gateway return `502`, or `504` on a timeout, when ServiceNow cannot be reached or reports an error; a create or update with no returned record also fails. The proxy request log now uses the `ims-gateway` field names, records every request, and is emitted in the shared server's structured text format (`INFO: request completed { ... }`) instead of the previous Echo-based format. Update parsers or monitoring that depend on the old log format. A response to a rejected token or a panic is now in the gateway's JSON format. Set the proxy's ServiceNow timeout with `servicenow.timeout`, as `proxy.timeout`, previously read from the wrong section, is only used if it is not set. The IMS Gateway uses `snow.timeout`. Update anything that depends on these.

* **Failures now fail:** `servicenow2 client`, `geneos incident raise` and `geneos incident query` now exit with a non-zero code, and an error message, in cases that previously exited with zero, such as a missing or invalid configuration file, no proxy or gateway URL, or a response that does not report success. Review Geneos actions and scripts that depend on the exit code. Only a proxy or gateway that cannot be reached, or reports `502`, `503` or `504`, results in the next configured URL being tried, and `geneos incident raise` now sends to only the first that accepts the incident.

* **Incident correlation:** The `servicenow2` proxy and `ims-gateway` now reject a request with no `correlation_id` (a `400` error), because the lookup for an existing incident would otherwise match any active incident for the configuration item. `geneos incident raise` now sets `correlation_id` from `__incident_correlation`, as documented, as well as the older `__snow_correlation`; if your `ims.yaml` sets `__incident_correlation`, incidents raised by earlier versions have no correlation ID and will not be updated by newer ones. For `servicenow2` set `correlation_id` or `_correlation_id` in the client configuration.

  ServiceDesk Plus also requires `correlation_id` or `__incident_correlation`. Correlation hashes from `ims.CorrelationID`, used by the IMS Gateway and `geneos incident`, are now lowercase hexadecimal, matching the ServiceNow clients. Records created with an older uppercase hash will not be found automatically.

* **CMDB tables:** The proxy's `_cmdb_table` and IMS Gateway's `__snow_cmdb_table` now default to `cmdb_ci` and accept only table names containing letters, digits and underscores. Values containing paths are rejected with `400`. Configure the intended table explicitly if you depended on the old missing-value behavior.

* **Literal request data and transformations:** Incident field values and ServiceDesk Plus GET queries are no longer expanded as configuration. Move any intended expansions into trusted configuration templates. `__` fields are excluded from ServiceNow payloads even without a transformation, but remain available to response templates. Dependent transformation renames now use the original fields rather than a random order, and invalid filter patterns fail instead of silently removing fields.

* **Redirects and credentials:** ServiceNow/proxy clients no longer follow HTTP redirects; configure final API URLs directly. Other affected HTTP clients keep credentials and OAuth tokens on the original scheme, host and port and reject HTTPS-to-HTTP redirects. ServiceDesk Plus tokens are read only from the user configuration directory, not the working directory.

* **Older `servicenow2` clients:** An earlier `servicenow2 client` still works with the new proxy, as the requests and responses are unchanged, and a client that sets `correlation_id` or `_correlation_id` is unaffected by the correlation check. However, it does not have the new error handling: it ignores any `4xx`, `5xx` or other failure response from the proxy, including the new `400`, `502` and `504` errors, and exits with a zero code and no message, as it did before. Upgrade the clients, as well as the proxy, to be told when an incident could not be created or updated.

* **ServiceNow Basic Auth restriction:** Instances where ServiceNow's Basic Auth restriction is enabled (`glide.authenticate.basic_auth.restriction.enforce`) reject the proxy and gateway with `401 User is not authenticated` unless their ServiceNow user has the `snc_basic_auth_api_access` role, even though the same user and password work in the web interface. Grant the role or use OAuth.

### Release contents

* **Removed from releases:** The original `servicenow` integration, `pagerduty` and `san-config`, together with their configurations and documentation, are no longer shipped. Use the versions in v1.30.1 or build them from source. For ServiceNow, consider moving to `servicenow2` or `ims-gateway`; both remain in releases.

* **Release archive includes:** The main archive no longer contains `/etc/geneos/includes`. Use the separate versioned `geneos-includes` archive or the embedded files managed by `geneos includes` instead.

### Email, webhooks and exported reports

* **SMTP authentication and ports:** SMTP selects the strongest offered authentication mechanism unless `_smtp_auth` selects one. Unencrypted authentication with `_smtp_tls=none` requires an explicit `plain-noenc` or `login-noenc`. Port 465 selects implicit TLS unless TLS is disabled. A failed custom-port connection no longer falls back to port 25 without TLS, and an explicit port 25 is no longer changed to 587. Invalid sender addresses fail.

* **Teams webhooks:** `libalert` and `libemail` require HTTPS URLs whose entire host is a supported Microsoft host; previously accepted lookalike or non-HTTPS URLs are rejected. `libalert` uses milliseconds for `_TIMEOUT`, matching `libemail`, and alert values are no longer expanded as configuration. No accepted webhook now means failure rather than success.

* **`dv2email`:** Filter lists keep items matching any pattern rather than requiring every pattern. Explicit output filenames no longer gain timestamps; only `auto` names do. The unused `--inline-css` option is hidden and ignored, serial numbering starts at 1, and entity-split output is ordered by entity name. Review consumers relying on the previous output.

* **Report formatting:** Formatted reporters sort by the first column only, keeping input order for ties. Previously ineffective scrambling, headlines and writer options now apply. XLSX updates replace the table rather than append, and duplicate first-column values no longer discard rows.

### Configuration, remote execution and Go libraries

* **Configuration expansion:** Values from lookups, environment, files, URLs, expressions and decoded secrets are inserted literally, without a second expansion pass. Configuration references and `Prefix` results still expand recursively, up to `Loop` (default 10); use `PrefixVerbatim` for untrusted prefix results. `$${` now escapes a literal `${`, lookup tables take precedence for all names, and self-references remain unexpanded instead of looping. Rewrite configurations that depended on the old expansion order or re-expanding external values.

* **Defaults and key/value parsing:** `UseDefaults(false)` disables all defaults, including `WithDefaults` and `DefaultsFrom`. Precedence is explicit settings and environment, main file, defaults files, embedded defaults, then `DefaultsFrom`. `SetKeyValuePairs` now interprets `key==value` as the value `=value` instead of ignoring it. Review callers relying on the old behavior.

* **File permissions and saved settings:** Credentials files and their new directories use `0600`/`0700`, and keyfile directories use `0700`. Ordinary configuration writes still include defaults and environment-derived settings unless filtered; they retain `0644`/`0775` defaults, configurable with `FileMode`/`DirMode`. Unreadable or corrupt existing keyfiles now fail rather than appearing usable.

* **Remote commands:** SSH arguments, environment and working directories are now quoted literally, so shell expansion of `$`, backticks and backslashes no longer happens implicitly. Remote processes use the remote home rather than inheriting local `HOME`/`USERPROFILE`. Remote Windows `Start`, `Run` and `Uname` return unsupported errors rather than trying Unix commands. Review commands that relied on shell expansion or local environment values.

* **Certificate APIs:** The `ecdh` key-type name now generates an ECDSA P-256 signing key, parsed as `ecdsa`. Private-key readers and writers normalize PKCS#1 and SEC1 keys to PKCS#8; malformed and encrypted PEM keys are rejected or skipped by `DecodePEM`. Invalid root CAs are rejected, unreadable nonempty stores are never replaced, and `ChangePassword` fails if a key cannot be decrypted. `Template` returns `nil` for invalid options instead of dropping them; use `NewTemplate` for an error.

* **Gateway Hub API:** `gwhub.EntitiesMetricsRows` now takes `EntitiesMetricsRowsRequest` and returns `EntitiesMetricsRowsResponse`; use the deprecated `EntitiesMetricsRowsLegacy` temporarily if needed. `FormatDuration(0)` returns `PT0S`, and `ParseDuration` rejects years, months and empty durations such as `P`.

* **Geneos XML model:** `fkm.File.Alias` is now `*SingleLineStringVar`, `sqltoolkit.DBConnection.Password` is `*Value`, and the processes plugin's `Parameter` uses `Name` for element text and no longer has `Value`. `StringList.UnmarshalText` and `NameValueList.UnmarshalText` reject nonempty text; `common.NewValue` joins slice elements with commas and returns `nil` for `nil`. Update Go callers constructing these types.

* **Network APIs:** `streams.RESTStream` now verifies HTTPS certificates; self-signed endpoints require a trusted certificate or explicit `streams.SkipVerify(true)`. Requests default to a 30-second timeout, and XML-RPC API responses to 60 seconds, configurable through their timeout options. REST non-2xx responses return typed `HTTPError`; redirected POST/PUT requests return `RedirectError`. Unimplemented Geneos REST methods return unsupported errors instead of success.

* **Process and logging APIs:** `process.PID` requires all specified arguments and returns the lowest matching PID. `SignalProcess` rejects nonpositive PIDs. Background `Start` returns the PID, foreground starts return zero, and `Batch` returns the channel used to stop child reaping. `logger.RedactSlice` no longer changes its input, and `Init` no longer sets global `color.NoColor`; use the handler's `Color` option. Rebuild downstream Go consumers and review these changed contracts.

## [v1.30.1](CHANGELOG.md#version-v1301)

> [!NOTE]
> Built with Go 1.27.3.

v1.30.1 replaces v1.30.0.

* **Configuration keys:** Change nested-key references from `::` to `.`, for example `tls::verify` to `tls.verify`, in configurations and scripts. Special cases include domain-name and IP keys in remote-host configuration.

* **SAN and Floating Netprobe gateways:** The `gateways` configuration changed from an object to an array. Review and manually update affected instance configurations; when adding or removing gateways with `geneos set` or `unset`, supply the complete gateway list because the original list is replaced entirely on change.

* **Templates:** Run `geneos init templates` after upgrading to use the new SAN and Gateway template features. Back up and compare any customised templates first.

* **Java truststores:** Special support for external command-line truststores was removed for `webserver` and `sso-agent`. Configure additional truststores through instance `options`; component-specific configuration is unaffected.

* **Automation and configuration:** `geneos list` separates the base symlink and package version in programmatic output. Configuration expansion can now repeat, which can affect configurations relying on single-pass expansion. `geneos ps --files` and `--network` are mutually exclusive.

## [v1.29.0](CHANGELOG.md#version-v1290)

> [!NOTE]
> Built with Go 1.27.1.

This release was withdrawn; use v1.29.1 or later instead. `gateway-reporter` was retired from builds in favour of the Gateway's built-in reporting features.

## [v1.27.0](CHANGELOG.md#version-v1270)

* Back up customised templates and compare them with the updated Gateway and Netprobe templates. For existing installations, run `geneos init template` after upgrading. The CHANGELOG also recommends `geneos backup --shared --aes --tls` for a rollback option.

* `geneos tls info` columns, names and ordering changed; update output consumers. Remote process checks now only see processes owned by the configured user, which can affect detection by `ps`, `start` and `stop`.

## [v1.26.1](CHANGELOG.md#version-v1261)

> [!NOTE]
> Built with Go 1.26.2.

v1.26.0 was withdrawn. The v1.26 series contains significant TLS changes:

* Existing TLS configurations continue to work, but using the new certificate and trust-chain features requires instance configuration changes. Read the [TLS changes](tools/geneos/TLS.md) and back up instance configurations before `geneos tls migrate`: migration is not reversible. TLS is enabled by default for new `init` and `deploy` installations unless disabled.

* Most `geneos` commands refuse to run as root without `--allow-root`; the global `allow-root=true` override is available but not recommended.

* To apply the corrected `listenip=none` default to existing SAN and Floating Netprobes, use `geneos set san listenip=none` and `geneos set floating listenip=none` for the affected instances.

* The `pkg/config` API has been refactored; external Go consumers must review removed and changed methods before updating their dependencies.

## [v1.24.1](CHANGELOG.md#version-v1241)

CSV reports no longer include the report title as their first line. Check consumers that depend on the previous output layout.

## [v1.24.0](CHANGELOG.md#version-v1240)

GDNA's `csv` format is now standard CSV, separate from `toolkit` output. Plugin lists in reports changed from `name(N)` to `name:N`, with spaces between plugins. Update report consumers as needed.

## [v1.22.0](CHANGELOG.md#version-v1220)

> [!NOTE]
> Built with Go 1.24.5.

The `cordial-geneos.xml` include was renamed to `itrs.level0.geneos-cordial.xml`; check references and diagnostics consumers. Previously deprecated command flags were removed. The reported local hostname changed from `localhost` to the real hostname, although both remain aliases.

## [v1.21.0](CHANGELOG.md#version-v1210)

> [!NOTE]
> Built with Go 1.24.3.

* `geneos ps` no longer has `--nolookup`/`-n`; `-n` now means `--network`. Port-list separators and table formatting also changed.

* Probe instances have a new `listenip` parameter. Remove any custom `-listenip` from an instance's `options` for the parameter to take effect.

## [v1.20.0](CHANGELOG.md#version-v1200---codename-leo)

Upgrade the GDNA binary before using the new Monitoring Coverage and Version dashboards. The old dashboard works with the new binary.

## [v1.18.0](CHANGELOG.md#version-v1180)

> [!NOTE]
> Built with Go 1.23.2.

* GDNA deprecated the original external filtering files; import their data using the new filter commands. Some YAML report options moved, including `enable-for-dataview` and `enable-for-xlsx`; update affected configurations. `gdna list` now uses subcommands such as `gdna list reports`.

* `geneos package install` defaults to component types already present; specify a type or `--all` to install others.

## [v1.17.1](CHANGELOG.md#version-v1171)

The Gateway `insecureport` parameter requires an updated `gateway-instance.setup.xml.gotmpl`; update it manually or use `geneos init templates`, taking care with customised templates.

## [v1.17.0](CHANGELOG.md#version-v1170)

`geneos package uninstall` uses `-F`, not `-f`, for `--force`, and `geneos package update` no longer supports `--install`. Local archive file paths now ignore `--version`; review installation scripts.

## [v1.15.1](CHANGELOG.md#v1151-changes)

Saved global configuration defaults may need to be removed manually to pick up new defaults. `geneos set x+="y"` now inserts a space only when the appended value begins with `-`. `geneos init san` and `init floating` were deprecated in favour of `geneos deploy san` and `deploy floating`.

## [v1.14.0](CHANGELOG.md#v1140-changes)

> [!NOTE]
> Built with Go 1.22.4.

The `geneos deploy` shorthand `-n` was repurposed for `--nosave`; review scripts using it for `--name`.

## [v1.13.2](CHANGELOG.md#version-v1132)

> [!NOTE]
> Built with Go 1.22.3.

v1.13.2 replaces v1.13.0 and v1.13.1. Review the [full breaking changes](CHANGELOG.md#v1132-breaking-changes):

* TLS flags in `init` and `deploy` changed: `--tls`/`-T` enables TLS, `--signing-bundle`/`-C` supplies signing material, and `--instance-bundle`/`-c` supplies instance material. The old long forms `--secure` and `--makecerts` remain aliases; `--importkey` was deprecated. Check existing scripts, especially their `-C` and `-c` usage.

* The shorthand for `--override` changed to `-O` in commands supporting it. The behaviour and flags of `aes new`, `aes import` and `aes set` changed; review their help before reuse.

* `package uninstall` requires `--update`/`-U` to update instances using older releases; without it, referenced releases are protected.

## [v1.12.0](CHANGELOG.md#version-v1120)

Go consumers of `pkg/config` should review the changed `LookupTable()` options; `LookupTables()` was deprecated.

## [v1.10.4](CHANGELOG.md#version-v1104)

`geneos aes import` no longer automatically updates instance key files; pass `--update` when that behaviour is intended.

## [v1.10.0](CHANGELOG.md#version-v1100)

`pkg/config` no longer embeds Viper; it is now named `Viper`, and not all previously promoted Viper methods are available. Review dependent Go code and use the package's mutex-protected wrappers where available. `Sub()` now returns an empty Config rather than `nil` for a missing key.

## [v1.8.0](CHANGELOG.md#version-v180)

* `geneos tls import` syntax and functionality changed; review existing scripts.

* External key files are enabled by default for new Gateways on GA5.14 and later. Existing Gateways are unaffected by the new default. Before changing key-file use for a Gateway that has already started and created a cache, follow the [Gateway key-file instructions](https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm#How_to_change_the_key_file_of_your_Gateway).

## [v1.7.0](CHANGELOG.md#version-v170)

`help` now shows the long description and usage; `--help`/`-h` shows the short description and usage. `pkg/config` also had API changes; Go consumers should review the CHANGELOG.

## [v1.6.4](CHANGELOG.md#version-v164)

Go code using `config.Plaintext` should review the change to `*config.Plaintext`.

## [v1.6.1](CHANGELOG.md#version-v161)

Gateway configuration output structures in `pkg/geneos` were renamed with an `Out` suffix down to `SamplerOut`; update dependent Go code.

## [v1.6.0](CHANGELOG.md#version-v160)

* Netprobe types moved under `netprobe/`. Existing installations should continue to work; `geneos migrate` can merge their directories and update configurations. Check customised SAN and Floating Netprobe templates for name clashes in the shared template directory.

* Some `pkg/config` `Load()` and `Save()` options were renamed without backwards compatibility; update dependent Go code.

## [v1.5.2](CHANGELOG.md#version-v152)

TLS root and signing certificate/key directories changed, and functionality previously in `tls new` moved to `tls create`. Review scripts and paths.

## [v1.5.0](CHANGELOG.md#version-v150)

* Check SAN gateway configurations if connections fail; the gateway configuration format changed. Re-setting the gateways may help, but manual cleanup of `san.json` may still be necessary.

* SAN and Gateway variable storage changed. Old values are converted when the configuration is updated, but previously incorrect variable-name casing is not corrected automatically; review it.

* Running instances as other users or under `sudo` was deprecated. Do not rely on the old privilege-handling behaviour.

* `pkg/logger` was removed, and `pkg/config` and other public APIs changed. Update dependent Go code before upgrading.

## [v1.4.2](CHANGELOG.md#version-v142---20221221)

Remote hosts named by IP addresses are renamed to `A-B-C-D`; check scripts that refer to those names.

## [v1.4.1](CHANGELOG.md#version-v141---20221219)

The `pkg/config` expansion functions gained options, changing their API. Go consumers should update calls as necessary.

## [v1.2.1 release candidates](CHANGELOG.md#version-v121-rc3---20221007)

The release-candidate changes leading to v1.2.1 included removal of built-in credential opaquing in output in favour of `${enc:...}`. Existing instance configuration files are not automatically updated to use configuration expansion; see the [rc1 notes](CHANGELOG.md#version-v121-rc1---20220928). Support for `$var` expansion was removed in favour of `${var}` in [v1.2.0-rc2](CHANGELOG.md#version-v120-rc2---20220926).

## [v1.2.0-rc1](CHANGELOG.md#version-v120-rc1---20220921)

The Geneos `api` and `api-streams` XML-RPC package APIs were substantially changed. Other public package APIs also changed since v1.0.0; review dependent Go code.
