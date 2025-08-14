# Change Log

## Version v1.22.3

> [!NOTE]
> **Released 2025-08-14** - Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/)

### Version v.1.22.3 Fixes

* `tools/geneos`

  * `restart` was broken by the earlier implementation of a process cache. Add a `GetLivePID()` function to reset/bypass the cache

### Version v.1.22.3 Changes

* Update Go to 1.25.0 and update dependencies.

---

## Version v1.22.2

> [!NOTE]
> **Released 2025-08-13** - Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/)

### Version v.1.22.2 Fixes

* Where we use a custom `http.Transport`, and it's appropriate, add a `Proxy: http.ProxyFromEnvironment` field setting to restore proxy support. In some components, such as the `servicenow2` integration this broke connectivity where the previous version used the `http.DefaultTransport` round-tripper.

* `tools/geneos`

  * Fix CurrentVersion() to return a valid value. This fixes version selection for component command lines as well as `geneos ps` output.

* Fix `cordial.LogInit` to ignore the current `zerolog.CallerMarshalFunc` value rather than just testing for nil. This means our custom function actually works.

* `integrations/servicenow2`

  * Fix various bugs and irritations based on feedback from beta testing.
  
  * The `query` command has new flags and can accept a full Service Now style query for incidents rather than just a username.

### Version v1.22.2 Changes

* Update to use Go 1.24.6

* `tools/geneos`

  * Add a process cache with a TTL of 5 seconds so that we don't have to iterate through `/proc` (on Linux) for each instance. The 5 second TTL is long enough for each execution. TODO: also cache per-process file and network entries.

---

## Version v1.22.1

> [!NOTE]
> **Released 2025-08-01** - Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/)

### Version v1.22.1 Changes

* Added tests to various packages and other parts based on cases created by Cursor, but hand-reviewed and edited to suit

* `tools/geneos`

  * updated documentation using Cursor

  * work on making Windows native support work, starting with Active Console. You can install and run AC2 instances using `geneos.exe` once you have configured the environment; this lets you manage versions easier that juggling ZIP files and directories

### Version v.1.22.1 Fixes

* `integrations/servicenow2` - add missing code to load table defaults on the proxy side. This fixes missing default values for fields not set on client side.

* `pkg/config` - fix string expansion edge-cases identified by newly added tests

* `tools/geneos` - fix minor bugs found during other work

---

## Version v1.22.0

> [!NOTE]
> **Released 2025-07-29** - Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/)

### Version v1.22.0 Highlights

* `tools/geneos` - New `backup` and `restore` commands to help you move and duplicate instances between servers. Originally developed to help set-up standby servers for Geneos Gateways, these commands can be used to make clean backups of one or more instances, selectively including sensitive files like AES key files and certificates with private keys. These can then be restored, optionally remapping instance names.

### Version v1.22.0 Changes

* Update Go to 1.24.5

* `tools/geneos`

  * Changes to the earlier `cordial-geneos.xml` include file
  
    * Renaming to `itrs.level0.geneos-cordial.xml` to better align with upcoming projects

    * Adding more information in the diagnostics views as well as using more conventional row names for the socket view

  * Changes to the output for the hostname from `localhost` to the real hostname, but all commands alias `localhost` and the hostname of the local server together

  * Removed previously deprecated command flags

### Version v.1.22.0 Fixes

* `tools/geneos`

  * Various fixes to spelling errors, both internal and external

  * Fixes for the Windows executable to start support local instances. This still needs more work, but it is possible to initialise a directory layout, install packages and start a local Netprobe and Active Console. Work to address the many remaining issues continues.

  * As part of the new `save` command implementation the lists of cleaned and purged patterns has been reviews for all component types

  * `geneos aes set --update` will not fail trying to backup a key file that does not exist

---

## Version v1.21.0

> [!NOTE]
> **Released 2025-06-25** - Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/)

### Version v1.21.0 Highlights

* A new [ServiceNow incident integration](integrations/servicenow2/README.md)

* A [cordial-geneos.xml](tools/geneos/includes/cordial-geneos.xml) include files that leverages the new `--toolkit/-t` flags for the various list subcommands, to monitor the cordial `geneos` managed environment.

### Version v1.21.0 Changes

* Update Go to 1.24.3 and update all dependencies

* Build all shared libraries in a UBI8 (RHEL8) container to maximise compatibility with older GLIBC versions

* Update `mapstructure` imports to support new Viper version and their fork of mapstructure

* `tools/geneos`

  * Add `--toolkit/-t` flag to all list command (as well as `ps`) to output information about the managed environment and more.

  * Enhance the `ps` command to support listing of open files (`--files/-f`) and TCP connections (`--network/-n`) for matching instances.

  * Additional changes to `ps` command:
  
    * Removal of the `--nolookup/-n` flag as we now cache user/group name lookups and also re-use the `-n` flags as the short form for the new `--network` output format.

    * Change separator character for port lists in CSV / Toolkit output from colon to space

    * Change separator for listening ports in table mode to a comma and remove surrounding brackets

  * `geneos package ls` output is now sorted by host and component name.

  * Do pre-start checks for all files (that must exist) before attempting to start an instance. Report any missing files to the user but continue starting other instances. Files that are created by components, such as logs, are not checked for and neither is the permissions to create those files.

  * Add a new `listenip` configuration parameter to probe types, which defaults to the string value `"none"`. If you have probe instances with a custom `-listenip a.b.c.d` in your `options` parameter then you must remove this before this new parameter will take effect.

* `libraries/libemail.so`

  * Support variable substitution in `_SUBJECT` and other subject settings
  * For Teams support, make the validation of URLs a list of regexes and add an `azure.com` pattern

* `pkg/config`

  * Add support for backslash escapes and escaped closing braces to `Expand*` functions. This was added for the new ServiceNow v2 integration.

* `pkg/xmlrpc` (and `gdna`)

  * Fix a possible file descriptor leak for long running processes by closing idle HTTP client connections after use.

### Version v1.21.0 Fixes

* `gdna`

  * Update dashboards, both in appearance but also functionality. The previous versions were missing some modifiers.

* `tools/dv2email`

  * Check for errors, and present them, from the `AttachHTMLTemplate` function

* `tools/geneos`

  * `geneos tls renew` should not create new certificates for instances that did not previously have them

  * Fix importing of root CA cert when importing a signing bundle

---

## Version v1.20.1

> [!NOTE]
> **Released 2025-01-16** - Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/)

### Version v1.20.1 Changes

* Update Go to 1.23.4 and update all dependencies

### Version v1.20.1 Fixes

* `tools/geneos`

  * Fix handling of older package directories that have `GA` or `RA` prefixes and sort packages by the numeric part only using semantic versioning rules.
  * Don't stop if there is no private key in a TLS bundle, it may be supplied in a separate argument
  * Do not overwrite existing root and signing certs and keys if they exist when running `geneos tls init`
  * Report if public key is missing or changes in user's `known_hosts` file
  * Always try to create the user config directory when writing configurations

* `gdna`

  * Remove dependency on license token when listing unused plugins
  * Use timestamp of detailed report when updating summary report
  * Update include file with additional configuration

---

## Version v1.20.0 - Codename "Leo"

> [!NOTE]
> **Released 2024-12-13** - Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/)

> [!TIP]
>
> This release is dedicated to my best boy Leo, who left us in late November 2024. In his 14 years he wasn't just a wonderful companion but he also helped, with his also now departed brothers, to save the lives of many other cats by donating blood through the RVC Animal Care Trust. For those reading this (and are near to Potters Bar in the UK) who believe their cats or dogs could become blood donors, please see <https://www.rvc.ac.uk/small-animal-vet/specialist-referrals/for-pet-owners/pet-blood-donations>
>
> <img src="screenshots/1311_BSHclients_181.jpg" alt="Leo, the best cat ever" width="500">

## Version v1.20.0 Changes

* `gdna`

  * Thanks to all the valuable feedback, we have a completely new Monitoring Coverage and Version dashboards. These new dashboards depends on the changes to some reports, so you **must** also upgrade the `gdna` binary before you can use it. The old dashboard will continue to work with the new binary but we may remove some of the data driving it from reports in future releases.
  * Add a new report `multiple-os-versions-per-hostid` which shows all servers that have more than one OS version for associated probes. This is an indication of invalid (duplicated) host IDs.

* `pkg/host`

  * Update <golang.org/x/crypto> dependency to mitigate potential issue with `ServerConfig.PublicKeyCallback` (CVE-2024-45337)

* `tools/geneos`

  * Added support for SSH private key files to the `geneos host` subsystem
  * Added a `geneos host unset` command to work with the changes require for private key files and more

* `integrations/servicenow`

  * Expose `incident-query` configuration to allow custom lookups
  * Some code clean-up and refactoring
  * Add example of using an encrypted API key between client and router

## Version v1.20.0 Fixes

* `gdna`

  * Fix `server-groups` report performance by joining on gateways as well as servers in the SQL query
  * Trim spaces from Gateway names and also update any existing entries to remove leading/trailing spaces. This can happen because even though we trim each incoming CSV field the Gateway name is prefixed `gateway:` and can hide a leading space

* `tools/geneos`

  * Support Webserver > 7.1.1 byt removing unsupported Java command line option
  * More updates to support OS and ARCH in downloads
  * Always set `HOSTNAME` environment variable in Netprobe so that Collection Agent self-monitoring works without further updates (this was already the case for `ca3` components)

---

## Version v1.19.2

> [!NOTE]
> **Released 2024-11-14**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).
>
> This patch release replaces v1.19.1 which did not fully address the issue of downloading releases of aarch64 architecture.

## Version v1.19.2 Fixes

* `tools/geneos`

  * Fix (work around) downloading of RHEL9 releases for 7.1 and later that could download the newly added aarch64 files.

---

## Version v1.19.0

### Version v1.19.0 Highlights

* Added `sso-agent` to `tools/geneos` so you can now deploy and manage those easier
* Updated `ca3` unmanaged/standalone Collection Agent component to work with new releases


> [!NOTE]
> **Released 2024-11-11**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## Version v1.19.0 Changes

* `tools/geneos`

  * Added `sso-agent` as a component. You can now install and manage SSO Agent instances using `geneos`. When adding a new instance `geneos` also creates an `ssokey` in a keystore file based on the `conf/sso-agent.conf` settings. If the instance is created with a TLS certificate and key then these are also stored in the truststore/keystore alongside any certificate validation chain. At the moment the configuration file is treated as read-only, but in time we will add capabilities to make changes using the command line.
  * As part of the work to add ZIP file support for SSO Agent component releases, we've improved how all releases are unarchived and now files and directories have their modification times set based on the archive file data. Symbolic links only have their modification times set when unarchiving locally.
  * Support the import of truststore/keystore files as JCEKS if loading as a JKS fails. Note that all truststore/keystore files are still written as JKS.
  * Fixed `ca3` starting environment
  * Changed output of versions in the `geneos ps` command to be easier to read

## Version v1.19.0 Fixes

* `tools/geneos`

  * A recent change meant that the user's AES keyfile would not be created when running `geneos login` for the first time, this is fixed
  * Explicitly disable file descriptor inheritance in started processes
  * Show a version for Java bases components based on installed version
  * Fix `deploy` commands to create the correct Geneos home directory, instead of always a `geneos/` directory in the working directory

* `gdna`

  * Include gateways that have no other components listed, fixing reports of unused gateway.

---

## Version v1.18.2

> [!NOTE]
> This patch release **replaces** v1.18.1 to add a fix for webserver start-up from the `geneos` command
>
> **Released 2024-10-29**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## Version v1.18.2 Fixes

* `tools/geneos`

  * Fix webserver starting command for releases after 7.0.4 to account for valid host header configuration, defaults to `.*`
  * Fix webserver command line option ordering

* `tools/dv2email`

  * Fix example command XML to revert overzealous edit of `::` to `.`

* `gdna`

  * Restore the `OTHER` row to Gateway and server group reports
  * Add `dynamicEntities` column to `gateway-groups` report
  * Update Gateway include to publish a schema for the `GDNA Process` sampler
  * Allow for unauthenticated SMTP (when username is empty)

* `pkg/email`

  * Allow for unauthenticated SMTP (when username is empty)

---

## Version v1.18.0

> [!NOTE]
> **Released 2024-10-17**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## Version v1.18.0 New Features

* `gdna` - This is the third public release of GDNA and we've added new features and made changes based on your feedback. The full details are in the documentation, but the highlights are:

  * Customise Level 1, Level 1 Optional and Level 2 plugin lists, including being able to require multiple instances of a plugin.
  * Added a new filtering feature to **include**, **exclude** and **group** by categories such as Gateways, Servers etc. See [FILTERS.md](gdna/FILTERS.md) for more.

* [`pkg/reporter`](pkg/reporter)

  * The reporting code from `gdna` has been pulled out into it's own package so that we can reuse this for other projects later. The API is not yet final and there are missing features (such as severity colouring for HTML tables, like `dv2email` can do) but it should settle in the next couple of releases.
  * The XLSX reporter can include headlines in each sheet, their inclusion and direction can be controlled by an option

## Version v1.18.0 Changes

* Updated to use Go 1.23.2

* `gdna`

  * Lists of plugins, such as in *missing* columns and lists per server and probe, now show how many instances after the plugin name, if there is more than one 
  * Command line sub-commands added to manage the filters and a growing set of right-click Commands added to the shipped include file
  * The original support for external files for more limited filtering has been deprecated, so if you use these you will have to import the data using the new command line sub-commands
  * When running "split" reports (the Gateway Overview and Details) the program now removes Dataviews that are no longer in the set of splits
  * New reports have been added to list more actionable information about your estate and some existing reports have been enhanced
  * Dataview Groups have been reworked from the ground up to offer a more logical view of `gdna` data
  * Every report now have a `reportName` headline to allow for better right-click command automation
  * Some YAML options have moved to allow for more logical grouping, so if you use `enable-for-dataview` or `enable-for-xlsx` to control which reports are autogenerated then you will have to change these to the new layout, see the docs for more
  * Split the growing [README.md](gdna/README.md) file into a number of parts
  * Added sub-commands to `gdna list` so now `gdna list reports` is needed to see what you can run. Also added the Dataview Group to the output
  * Other "under the hood" changes
    * In the docker image the `gdna.yaml` file is now located in the `geneos` user's home are and not root directory
    * Allow `~/` as a prefix for the `db.file` config parameter, which is in turn pulled into `db.dsn`
    * Update default `db.dsn` to include `?cache=shared` to avoid common locking issues

* `tools/geneos`

  * `geneos package install` will now limit installed components to those already existing unless given either a specific type or the `--all` flag. This is intended to make downloads quicker as most users will only want to download component packages for instances that already exist.

## Version v1.18.0 Fixes

* `tools/geneos`

  * Set the default local directory for downloaded packages when using `geneos package install -L` without a file or directory path
  * Restore installs from Nexus for internal users
  * Fix `geneos package delete` to stop it removing the latest of each component package. Other fixes to `geneos package delete` include not removing packages which are only used by instances as the `pkgtype` (e.g. `minimal/netprobe`) and the command now also ignores text prefixes on version directories like `GA`
  * Fix handling of wildcards that match nothing in ParseArgs()
  * Fix regexp pattern that extracts release information to allow for old releases that used a `.` instead of `-` between version and architecture

---

## Version v1.17.1

> [!NOTE]
> **Released 2024-09-18**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## Version v1.17.1 Changes

* `tools/geneos`

  * Add an `insecureport` parameter in Gateway instance template to support explicit insecure port when secure port is enabled by certificates. To use this you must update the `gateway-instance.setup.xml.gotmpl` either manually or by using `geneos init templates`
  * Related to the above, swap the main and alias names for `geneos init template(s)` - both will work.

* `gdna`

  * Changed the text value in the `l3missing` column of various view from `yes` to `missing` to avoid ambiguity after user feedback.
  * Changed Dataview group name for optional gateway reports from `Coverage by Gateway` to simply `Gateways`
  * Add a new `logic` field to conditional-format tests to control how multiple column tests are handled; Default to the previous behaviour of "AND" but can be "OR" (or "all" / "any" respectively)
  * Add a `geneos.dataview-create-delay` duration to allow smoothing of dataview creation when creating many hundreds of dataviews like for split reports. Only applies when a dataview is created, not during updates to existing ones
  * Add a "Version Summary" dashboard using new Geneos 7 data
  * Revisit headline names and some column names for consistency and clarity
  * Add a dashboard slide show to the docker image web server configuration

* `tools/san-config`

  * Add support for number of gateway sets to push to each probe, default 1 to maintain existing behaviour
  * Treat gateway pairs as a single gateway in logging and counting
  * Connection files can be served without a standby gateway port
  * Use default ports when gateways in the configuration do not supply them

* `integrations/servicenow`

  * Add extra control over user name field with `incident-user` router parameters
  * Add a `trace` parameters to output all HTTP request and response data to stderr, for diagnostics
  * Clean-up some old code

## Version v1.17.1 Fixes

* `tools/geneos`

  * Fix `copy` and `move` handling of source or destination in the form of `@HOST` by updating command annotation and related changes in `instance.Match()` and `parseargs()`. Before this fix, and after the `parseargs()` review, `geneos copy gw\* @remote` would not work correctly, making the creation of standby gateway sets much harder

* `gdna`

  * Fix filtering of plugins, samplers and dynamic entities for multi-gateway reports, now numbers add up.
  * Fix table used for building `servers` reporting tables
  * Fix spelling of `latest` columns in the `sources` report
  * Fixes to some edge cases in probe and sampler counting
  * When an OS or Version is an empty string, convert it to NULL for later reporting as Unknown

---

## Version v1.17.0

> [!NOTE]
> **Released 2024-09-08**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

### v1.17.0 Changes

* `tools/geneos`

  * Add a `minimal` Netprobe flavour that can be added either as it's own component type or, more commonly, as a flavour of a normal Netprobe, e.g. `geneos add netprobe minimal:probeName` - this extends the way you install a Fix Analyser probe as a SAN, e.g. `geneos add san fa2:myFixSAN`
  * Add a `use-chain` parameter for all instances which defaults to `true`. Set this to `false` using `geneos set ... use-chain=false` to disable certificate validation using a configure chain file.

* `gdna`

  * Add support for GA7.0.0 updates to `licd` reports and additional details from corresponding Gateways. This includes OS and component version information as well as enriched data for dynamic entities.
  * Add schema updates for the new table columns. Rollback should work - untested at time of writing - as no existing columns have changes, just new ones added.
  * Add extra reports using the above data to a `Monitored Estate` groups, moving `Sources` to the same group.
  * Add new headlines for Gateway summary and detail Dataview and reports to show host, port and version - when available.
  * Add new columns to `Missing Coverage` and Gateway summary and details to expose new data when available and rename existing columns (not used in dashboards) to better align with each other.
  * Update `Sources` Dataview and report to report a status of `STALE` when a file is invalid. A remote source is always either up-to-date or inaccessible, which is a different status.

* `tools/san-config`

  * Initial commit of a SAN remote configuration file server that allows dynamic building of SAN XML files using an YAML based inventory as well as autodetection of live Gateways from a preconfigured list.
  * This tool is not yet in the main build as further changes and documentation are required for it to be more useful.  

### v1.17.0 Fixes

* `tools/geneos`

  * `geneos package` subsystem gets a wide review and a number of changes and fixes:

    * `uninstall` - change `-f` to `-F` for short form `--force` for consistency
    * `update` no longer supports the `--install` flag
    * `install` - local files now ignore the `--version` flag for file (non-directory) paths. For systems with a platform type (el8/el9) only install that platform type of non-platform specific archives. This allows components like web dashboard servers, which have no platform specific releases, to be installed.
    * `install` - many other changes around the handling of local files to improve what gets installed
    * Fix instance iteration so that only affected instances are updated and so on. 

  * Fix `migrate` and `set` and `config set` to actually work with changes to parseargs() function.

  * When adding components with key file support use the `${config:home}` value to pin key-file to working directories. Also, when saving a component configuration, such as during `move` or `copy` update the key-file and previous key-file paths to use `${config:home}` when source path matches the instance home directory. 

* `gdna`

  * Fix misuse of source tables when building active gateway table (no actual data change as the servers table has the same data)

* `pkg/config`

  * Update `Path()` to better reflect the likely sources used by `Load()`. Pass `config.MustExist()` as an option to `config.Path()` to get a result based on actual files rather than the likely file to be used.

* `integrations/servicenow`

  * Fix loading of configuration file in use of `config.Load()` options (explicitly use YAML)

---

## Version v1.16.2

> [!NOTE]
> **Released 2024-08-08**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

### v1.16.2 Fixes

* `gdna`

  * Update Monitoring Coverage dashboard to remove references to `Demo Gateway` and improve visibility of per-plugin vertical bars.

---

## Version v1.16.0

> [!NOTE]
> **Released 2024-08-02**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

### v1.16.0 Changes

* `gdna`

  * Updates to Dashboards to present more focused information to users
  * Improve documentation, add block diagrams and update screenshots
  * Added a `gdna version` command, when combined with `--debug` will report dependencies used during build
  * Move to using `github.com/mattn/go-sqlite3` instead of pure Go `modernc.org/sqlite`

* Build

  * Use Go 1.22.5

### v1.16.0 Fixes

* `gdna`

  * Fix some SQL queries used in temporary table construction to give more accurate numbers for servers with multiple probes
  * Fix dashboard XPaths to remove unnecessary Gateway and Managed Entity names

* `tools/geneos`

  * Fix `geneos package install` (and `geneos init` etc.) to have better support for local installation sources and to distinguish between el8/el9 platform specific archives over non-platform specific ones

* Build

  * Fix Makefile/Dockerfile interdependencies and speed up documentation builds by downloading Chrome outside of underlying puppeteer installation

---

## Version v1.15.6

> [!NOTE]
> **Released 2024-07-25**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

### v1.15.6 Fixes

* `tools/geneos`

  * Fix matching of names, which because we didn't use the return value of `slices.Compact()` meant an extra "all" was effectively added to all commands
  * Remove non-numeric prefixes of directory version naming to all "ra6.9.0" etc

* `gdna`

  * Fix counts for plugins over servers with multiple probes
  * Update docs to fix references to `geneos` which are meant to be `gdna`
  
---

## Version v1.15.5

> [!NOTE]
> **Released 2024-07-22**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

### v1.15.5 Fixes

* `tools/geneos`

  * Restore validity check regexp to allow dashes and underscored in remote host names
  * Merge parsed params back into args to fix `geneos host add` with an ssh URL

---

## Version v1.15.4

> [!NOTE]
> **Released 2024-07-17**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

### v1.15.4 Fixes

* `gdna`

  * Fix `sources` HTTP schema name, instead of using HTTPS
  * Remove `sources` entries when configuration changes or files are removed
  * Remove unused plugins from `expectRows` in include file
  * Fix plugin grouping for CA and GW samplers and resolve numeric inconsistencies in queries
  * Update bundled dashboards to include `Unused Gateways` and to add missing plugins

---

## Version v1.15.3

> [!NOTE]
> **Released 2024-07-11**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

### v1.15.3 Fixes

* `tools/geneos`

  * Fix handling of `-el8` and `-el9` package installs that were broken when updating for new Netprobe package file names
  * Fix matching of instances to properly account for wildcards and non existing hosts

* `gdna`

  * Keep `gateways_all` table, not as a potential temporary one. Fixes sources report when sources go stale amongst other things
  * Fix logic in `sources` report to evaluate and update status correctly when `valid` changes

### v1.15.3 Changes

* `gdna`

  * Add an `unused-gateways` report
  * Add headlines to `gateway-coverage` to make clear how many unused gateways exist

---

## v1.15.2

Replaced by v1.15.3 to fix a one character typo in a regexp.

---

### v1.15.1 Fixes

  * Fix handling of new Netprobe release archive names in 6.9.0

### v1.15.1 Changes

* `tools/geneos`

  * add support for a `minimal` component type, which is a Netprobe without Collection Agent
  * `geneos init demo` now supports the `--minimal`/`-M` flag to use the minimal Netprobe in the deployment
  * the global configuration file (typically `${HOME}/.config/geneos/geneos.json`) now only saves chasnges to defaults. This allows defaults to be updated in the code without needing manual changes. Note that you may need to edit the file to clear out saved parameters that were the default but have the defaults have now changed.
  * changes `geneos set x+="y"` behaviour such that a space is only prefixed to the new option if it starts with a dash (`-`). This allows paths and other parameters that are not space separated to work
  * the `geneos init san` and `geneos init floating` commands have been deprecated in favour of `geneos deploy san` and `geneos deploy floating`, respectively
  * add the start of a `geneos config export` command that archives instances either as a back-up or for re-import later, once the matching `geneos config import` command is added. This is work-in-progress.

* `gdna`

  * Update to Geneos 6.9.0 and use the `minimal` Netprobe to reduce image size
  * Add a `gdna stop` comma\nd, to stop background GDNA processes. An initially empty `gdna restart` command has been added for later completion.

* General

  * Centos 7 has gone EOL so no test images can be built, remove them

---

### v1.15.0 Changes

> 🎁 Introducing GDNA 🎈
>
> GDNA (Geneos Dynamic Netprobe Analysis) gives an overview of the levels of Geneos monitoring coverage in your I.T. estate and prompts action to ensure that there are no gaps in the visibility of the health of your applications and systems. It does this through analysing Geneos license allocation data and applying heuristics to plugin usage information to present visual indications of coverage levels through two dashboards.
>
> For many users installing GDNA is as easy as starting up a docker container after configuring the data sources.
>
> See the [`gdna`](gdna/README.md) README for more details.

* General / Miscellaneous

  * Move to Apache 2 license, from MIT, for clarity and conforming to ITRS company guidance
  * Move cron Centos 8/9 test images to UBI 8/9 respectively
  * Change all warning triangles in docs and comments ("⚠️") to to more friendly bulb ("💡")

* `tools/geneos`

  * Refactor internal routines to iterate over instances. Reduces the number of similar functions and inter-dependencies.
  * Refactor other function names for clarity

* `pkg/config`

  * Add a `UseKeyFile()` option to expand functions to allow overriding the key file to use

### v1.15.0 Fixes

* `gdna`

  * Fix broken modifiers that result in no data visibility with updated sampler name
  * While fixing above also fix references to old `t1` column to use `100` to indicate complete coverage
  * Fix logfile selection from command line

* `tools/geneos`

  * Fix the incorrect selection of archive directories in `package install` when using the `-L` option
  * Fix the `--nosave` option by resetting error in return to nil, as it was a "valid" failure

* `pkg/config`

  * Fix use of "internal defaults" in Load() when using the SetConfigFile() option

* Documentation

  * Fix emoji in generated PDF files in Dockerfile by adding an appropriate font
  * Fix missing screenshots in markdown files exported to releases by merging all screenshots directories and shipping that too. PDFs not affected.

---

### v1.14.0 Changes

* Update Go toolchain to use 1.22.4

* `tools/geneos`

  * Add options to `tls create` to specify destination directory, outputting certificate bundles and setting the expiry period in days. Also, do not initialise the TLS subsystem if it's not found unless the `--force` flag is used.
  * Hide `--name` option in `deploy`, repurpose short-form `-n` for new `--nosave` option
  * Rename `logs` option `--nostandard` to `--no-stdout` (with alias) for consistency

* `cordial` package

  * Only log file and function if logging at `Debug` level or greater
  * Split logging methods into their own file, add options to `LogInit()` to specify filename and/or [`lunberjack`](https://pkg.go.dev/gopkg.in/natefinch/lumberjack.v2) settings. Log filenames can also include `~/` prefix for home directory.

* `tools/gateway-reporter`

  * Add an optional XLSX password in `output` config section

### v1.14.0 Fixes

* `tools/geneos`

  * Fix download base name for Netprobes to new value.
  * Fix `deploy` importing of keyfiles (only import if given on command line)
  * `package install` and `package update` - Fix the handling of package metadata (e.g. `+el8`) suffixes to that they are treated the same as a package without a metadata suffix - and the right one is selected for the system being installed

* `pkg/config`

  * Better recovery on failing to find the user's config directory, return an error instead of panic-ing

---

## Version v1.13.2

> **Released 2024-05-23**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

**💡️ This release replaces v1.13.0 and v1.13.1 and addresses two issues found after release, one for `dv2email` not authenticating with all SMTP servers correctly and changes to `geneos package uninstall` to protect against unintended updates**

💡️ In addition to the breaking changes below please note that some documentation is still out-of-date, and the details in this changelog are more likely to be up-to-date.

## v1.13.2 BREAKING CHANGES

💡️ There are a significant number of **BREAKING CHANGES** in this release both for the `geneos` program, specifically around TLS and AES command line options and behaviour, but also for some of the `pkg` APIs.

* `tools/geneos`

  * `tls` Subsystem

    After reviewing the usability of the TLS subsystem commands to manage secure connections with users, we've re-worked some of the functionality and added a new `tls export` command. The command line flags for the `tls import`, `deploy` and `init` command have been revisited and an attempt been made to align them more closely. Now, the following flags have common meanings:
    
      * `--tls`/`-T` - enable TLS secured connections. This applies to all the `init` commands and the the `deploy` command. Was previously `--secure`/`-C` in `deploy` and `--makecerts`/`-C` in `init` commands. Both long forms are aliased to the new flag.
      * `--signing-bundle`/`-C` - specifies the source of signing certificate and private key and optional verification chain. This is new to `deploy` and replaces some of the functionality of `--importcert` for `init` commands where the short form has changed case for consistency.
      * `--instance-bundle`/`-c` - specifies the source of an instance certificate and private key and optional verification chain. This is new to `deploy` and replaces the rest of the functionality of `--importcert` for `init` commands.

    Other flags have been deprecated, such as `--importkey`/`-k`, whose functionality has been merged with the above.

    The new `tls export` command has been added to help extract those parts of the local TLS configuration signing certificate and private key that you need to push to a remote server to be able to create and manage certificates with a common trust relationship. Use the output of `tls export` with the other commands above and their `--signing-bundle` flags to do this.

    What this means for you is that if you need to manage Geneos instances across multiple servers (remote probes, standby Gateways etc.) but cannot use the existing SSH `host` features, then you can more easily keep the installation in sync for TLS certificates. Typical usage may be like this:

    On your main server run this, and copy the output to your clipboard (or output to a file with the `--output`/`-o` option):

    ```bash
    $ geneos tls export 
    --- BEGIN CERTIFICATE ---
    ...
    ```

    Then, log in to the new remote system and

    ```bash
    $ geneos deploy san -u downloaduser@example.com -C -
    [follow-prompts]
    Paste PEM formatted certificate bundle:
    [CTRL-V ENTER CTRL-D]
    ```

    Now your new Self-Announcing Netprobe will have a certificate created signed by the signing certificate and key you have exported on the central server.

  * `aes` Subsystem

    The AES subsystem was inconsistent and the functionality incomplete and, in reality, a mess. In this release we have reviewed the functionality of all of the AES commands. The support for AES keyfile creation, importing and management on remote hosts has been improved, but some issues may remain.

    The `aes ls`, `aes encode`, `aes decode` and `aes password` commands are largely the same while their internal implementations have been cleand-up. The other commands - `aes new`, `aes import` and `aes set` have all been rethought and their functionality and command line flags have changed. Please review the help text or documentation for full details. They should be more consistent and useful now, but as always please let us know if anything doesn't work as you would expect.

  * `deploy`

    As mentioned in the `tls` subsystem changes above, the command lin flags for the `deploy` command have changed, especially the `-C` and `-c` have been swapped around for consistency. Please note this if you have previously used `deploy` to create TLS protected Geneos deployments. The short form for `--override` is now `-O`.

  * `package` and `init` Subsystems

    The commands that support an `--override` flag have had the short form changed to `-O` to free the `-T` in `deploy` to be hort form for `--tls`.

* `pkg/config`

  The `config` package 

## v1.13.2 Changes

* Go dependency updated to 1.22.3

* `tools/geneos`

  In addition to the breaking changes, there have been a number of added features and functional improvements:

  * `package uninstall` now has an `--update`/`-U` flag to be explcit that the user wants to update instances which use older releases. Without this flag no release will be removed if it is referenced by an instance.

  * `webserver` instances will now automatically manage TLS though a custom `cacerts` file and a local keystore and `security.properties` file changes. This allows for both client and server TLS with trust chains. The instance chain file is added to the Java `cacerts` to add trust of other Geneos components, such as Gateways and SSO Agent while instance certificate and private key can be "real" PKI ones to offer a trusted TLS web interface.

* `cordial`

  * The top-level `cordial` package provides convenience functions, like `LogInit()`. This has been updated to not log source file and function unless the global log level if debug or trace.

* Use a new SMTP package - <github.com/wneessen/go-mail> - Swap out the old SMTP email package used to a newer, actively maintained one. The changes should not be user visible. The affected components are `pkg/email` and `tools/dv2mail`. There have been no changes to `libraries/libemail` as the changes required have not yet been tested and will be completed in a later release.

## v1.13.2 Fixes

Many of the changes above were prompted while tracking down and fixing issues around existing APIs and features in `geneos`.

* `tools/dv2email`

  * Fix the default HTML template to use the correct, new names for data identifying columns and rows.

  * Fix failure to authenticate to some SMTP servers.

* `tools/geneos`

  * Fix support for legacy `.rc` files, which was broken in a previous release.
  * Fix `tls ls -a` to show root and signing certificates even without instance certificates created.

  * Fix the handling of file paths on Windows with the `geneos.exe` binary.

* `pkg/config`

  * Fix the handling of file paths on Windows.

  * Use the newer `AbbreviateHome` / `ExpandHome` / `ExpandHomeBytes` functions to remove common code.

---

## Version v1.12.1

> **Released 2024-04-18**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.12.1 Changes

* General

  * Update Go used to 1.22.2

* `tools/geneos`

  * Add support for Collection Agent logs to `log` command
  * [#195](https://github.com/ITRS-Group/cordial/issues/195) - Add append capability to set command

* `pkg/rest`

  * Add a `SetupRequest` function to allow for custom HTTP client settings

* `pkg/config`

  * Fix a deadlock in `UnmarshalKey` wrapper
  * Add a `Bytes` method for encryption
  * Add `StopOnInternalDefaultsErrors()` option to bail when loading bad internal defaults

* `libraries/libemail`

  * Add a (deliberately undocumented) `_SMTP_TLS_INSECURE` environment variable to allow for edge cases in TLS testing
  * Build using Go DNS resolver and user lookup packages

## v1.12.1 Fixes

* `tools/geneos`

  * Fix `tls import` command and update docs
  * Deal with missing key files as errors in `aes` sub-system

* `pkg/host`

  * Allow no username for SSH config, which then defaults to the user running the command
  * Return an error from IsAvailable() on failure, and use this to send Dial() errors up the stack
  * Log a debug level message if Dial fails

* `cordial` (top level package)

  * Add support in `LogInit` for `-` and `/dev/null`/`NUL` as log destinations - STDOUT and discard, respectively

---

## Version v1.12.0

> **Released 2024-01-15**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.12.0 Changes

* `tools/files2dv` **New**

  A new program that can be run as a Toolkit (soon as an API sampler) to scan files and directories and create one row per file. The contents of files can be checked and extracted into columns - the difference between the standard Geneos FKM and Statetracker plugins and using `files2dv` is that the whole file is scanned until each column is either filled in or no matches found, in the latter case a failure value can also be substituted. This is not a line-by-line scanner, like FKM.

  If no contents are checked then just the matching file's metadata is used.

* `tools/gateway-reporter`

  `gateway-reporter` has had a number of issues fixed but also gets a new `csvdir` output format, which can be used to create a "live" directory of CSV files and an optional Gateway include file to read back reports into a Gateway-of-Gateways.

* `pkg/geneos`

  Added an `ExpandFileDates()` function to emulate Geneos expansion of file paths including `<today ...>` placeholders. Currently only `today` is supported.

* `pkg/config`

  Change `LookupTable()` options to take variadic args and deprecate `LookupTables()`

  Map dashes (`-`) to underscored (`_`) when using environment variable lookups, so configuration items like `file-name` can be references as `FILE_NAME` in an env var.

## v1.12.0 Fixes

* `pkg/geneos`

  Fixes around the handling of merged output files (where the `var-` prefix is dropped by the Gateway on some XML) so thet `gateway-reporter` works again.

* `tools/geneos`

  Fix `fa2` start-up to include `-secure` when in secure mode.

  Better handling of remote hosts and display of a Flags column to show if hosts have been "hidden" (disabled). While documentation needs to be updated, the difference between hiding a host and disabling it (which is not yet possible) is that a hidden host can still be referenced explicitly on the command line, but is not included in any "all" lookups, such as `geneos start`

---

## Version v1.11.2

> **Released 2023-12-07**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.11.2 Fixes

* `pkg/commands`

  - add custom UnmarshalJSON to Dataview type to record the received order of headlines, rows and columns

* `tools/dv2email`

  - use dataview ordering (above) to render dataviews as seen from the Gateway, matching the normal Active Console
  - add support to differentiate between unset and empty `--type` on command line

## v1.11.2 Changes

* `tools/geneos`

  - add support for secure arguments for fileagent releases 6.6 and higher

---

## Version v1.11.1

> **Released 2023-12-05**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.11.1 Fixes

* `tools/geneos`

  - Fix `ac2` start-up on Linux by passing on `DISPLAY` and `XAUTHORITY` envs to process, when they are defined

  - Output ISO8601 date/times in `geneos tls ls` outputs, not Go default format

---

## Version v1.11.0

> **Released 2023-11-23**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.11.0 Changes

* Major updates to [`dv2email`](tools/dv2email/README.md)

  `dv2email` has has a major work-over. In addition to fixing numerous bugs we've also added the ability to send dataviews as XLSX or Text Table attachments, splitting multiple Dataviews into either individual attachments or per-Entity. The `export` sub-command allows to you save Dataviews as files in XLSX, HTML or Text Table formats. Take a look and let us know what you think.

* New [`gateway-reporter`](tools/gateway-reporter/README.md)

  A new tool to build monitoring coverage reports from static Gateway configurations. This is a new tool and support is limited to a core set of plugins and the data reported for each. The tool does not interrogate running Gateways and so does not attempt to resolve dynamic parts of the configuration. Output is in XLSX, CSV and JSON formats.

  This is a new work in progress based on an internal project for a client and we welcome suggestions and contributions. Please use the links above to send us feedback!

* New example [`holidays`](tools/holidays/README.md)

  Another new tool, but very early in development. This uses the Python holiday module (which seems the only reliable, free source of global holiday data) to generate a list of holidays and their names in Go. It has to run in a docker container as the `cpy3` Go package only supports Python 3.7. It may prove simpler just to write the whole tool in Python and avoid Go, but the local skills are sadly lacking at the moment. Also, we have Go based tools in `pkg/geneos` to build Gateway configuration XML in a nice, clean way.

  The aim is to be able to create Gateway include files with predefined Active Times of holidays that can be incorporated into more general monitoring.

  Again, suggestions and, more importantly, contributions welcomed.

* [`geneos`](tools/geneos/README.md) has had a number of enhancements (and bug fixes, of course).

  The `geneos aes encode` command can now also create "app key" files for use with Gateway Hub and Obcerv Centralised Configs. To help with that the logic around how the Gateway command line is constructed has also had a refresh, with new options for the Centralised Config options. See the documentation for [`encode`](tools/geneos/docs/geneos_aes_encode.md) and the [`gateway`](tools/geneos/docs/geneos_gateway.md#centralised-configuration) for more information.

## v1.11.0 Fixes

* `pkg/`

  - Removal of all logging in packages. This was left over from early development and is considered bad practice.

* `tools/geneos`

  - Fix a mistake in handling remote hosts in `geneos show -V` for validation runs.

  - Fix `geneos home` matching of instances when the same name existed on local and remote hosts.

  - Fix issues in `geneos package` sub-system commands. They should now work more consistently and do what you expect.

---

## Version v1.10.4

> **Released 2023-11-03**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.10.4 Fixes

* `pkg/hosts` - disable sftp concurrent reads as this seems to mess with the file offset after io.Copy() and causes `geneos logs` to do strange things with remote log files

* `tools/geneos`

  * Fix a number of issues with `geneos logs`, including the results of the above but also how log file references are stored so that when you are tailing the same named instance on two hosts they get mixed up.

  * Update `aes` commands to be less error prone, change `import` to not automatically update key files in instances, but require a new `--update` flag. Update docs.


## Version v1.10.3

> **Released 2023-11-02**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.10.3 Fixes

* `tools/geneos` - Fix duplication of non-host qualified names introduced in last fixes - a string list was re-used instead of being locally allocated inside blocks.

* `pkg/config` - In Sub() copy the pointer to the mutex from the parent so that locks apply to the whole config object. Fixes concurrent access panic when methods are called on both the parent and the new sub-config. While here, also copy the other config fields, like delimiter.

---

## Version v1.10.2

> **Released 2023-10-31**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

💡️ The previous patch version was removed after finding the fix for wildcards was only partial. This second patch release addresses that issue.

## v1.10.2 Fixes

* `tools/geneos`

  * The recently introduced wildcarding for instance names was causing non-matching instance names to be evaluated all "all", so `geneos stop test` would stop all instances. Fix this by checking the list of names returned from the pattern match and if empty assume the input is not a pattern and leave it alone.
  * The fix to `pkg/host` below tries to ensure that running multiple sessions to remote hosts works more reliably, especially those where commands take some time to return, like for `start`

* `pkg/process` - Fix Daemon() for Windows by adding a `DETACHED_PROCESS` flag to new proc attributes.

* `pkg/config`

  * Protect global map access with a mutex in `expandoptions.go`
  * Make the viper mutex a plain one, not RWLock. Reorder calls to Unlock to cover decode()

* `pkh/host` - Allow for remote SSH session limits, retry NewSession() up to 10 times with a 250ms delay. This limit is in the remote server, so cannot be overridden.

---

## Version v1.10.0

> **Released 2023-10-25**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.10.0 Fixes

* `pkg/config`

  The `Sub()` methods will now return an empty Config struct and not nil if the key is not found. This is a divergence from viper.

  The non-global `GetSliceStringMapString` method returned values from the global config object. Found while implementing changes to noe use embedded viper, see below.

* `pkg/geneos/netprobe`

  Updates to the structs to produce valid XML when rendered as XML through the Go xml package.

* `tools/geneos`

  * Do not automatically try to `rebuild` component config files if the `setup` parameters is to a remote configuration.

  * Fix the `show -s` command to read instance configurations from their potentially remote host and not localhost.

  * Add a 250ms delay after starting an instance to allow for the process to fully start and update OS args so that the `GetPID` call works more often and can report the successful start-up.

## v1.10.0 Changes

* `pkg/config`

  Potential *API Changes* - to allow safer concurrent access to the underlying viper configuration objects the original embedded viper instance in the Config struct has been promoted to be named as `Viper`. This removes access to embedded methods and the intermediate methods have been updated to use a RWMutex around every call to viper. This however means that not all viper methods are transparently available and new shims have been added for the most common ones found. If dependent code now fails to compile because of missing methods they will need to be added to `config.go` along with the appropriate mutex wrappers.

  Added a `WatchConfig()` option to enable auto-reloading final config files found during `Load()`. Note that `WatchConfig` is not concurrency safe. This may change if we implement our own callback.

* `tools/geneos`

  * Add "glob" style wildcard support for instance names (and names only, not remote hosts) to most command. This should always be used with quoting to avoid shell expansion. This allows commands line `geneos start gateway 'LDN*'` and so on. Also add support to `move` and `copy` to act on multiple wildcarded sources as long as the destination is a `@HOST`.

  * Some instance configuration parameters are no tested for the instance `home` path and this is replaced with `${config:home}` so that moves and copies have paths auto updated. This include certificates, keys and set-up files.

  * Lower the auto-generated `instance.setup.xml` Gateway include file priority value so it is loaded before other typical includes.

  * For Gateway and SAN change default parameters `gatewayname` and `sanname` respectively to use `name` in an `GetString` expansion. This makes the parameters auto-update if the instance name changes (for example using `move` or `copy`) until and unless the user sets a fixed name.

  * Remove the `-setup-interval` from SAN command lines (which was using the default anyway) to allow it to be overridden in the `options` parameter.

---

## Version v1.9.2

> **Released 2023-10-10**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.9.2 Fixes

* `tools/dv2email`
  - Fix environment handling, which was broken in an earlier update to the `config` package
  - Add command line args for use from a Geneos Command
  - Update Dataview Row handling
  - Fix HTML template for multiple Dataviews
  - Update docs

## v1.9.2 Changes

* `pkg/config`
  - Add a `SetConfigReader()` option to Load to be able to load configuration from an io.Reader. Untested, work in progress for a project.

---

## Version v1.9.1

> **Released 2023-10-06**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.9.1 Fixes

* The release build process has been reworked to use Debian Bullseye images to maximise compatibility for shared libraries and also to build static binaries cleanly. A change in the Go toolchain at some point has made the build of the dynamically linked _centos7_ binary of `geneos` not work. This has now been removed while investigations into how to do this properly continue. This means that for users who have network directories for users there will be errors looking up users for `ls` and `ps` commands, at minimum.
* [`tools/geneos`](tools/geneos/README.md)
  - Use `path.IsAbs()` and not `filepath.IsAbs()` so that constructing paths on a Windows host works for remote Linux systems. Fixes process start from Windows to Linux.
  - Allow deletion of protected instances with the `--force`/`-F` flags, as intended originally
  - When creating instances check all listening ports, not just those reserved in instance configurations
  - More fixes to package handling around component types with parent types
  - Change TLS cert verification to validation and document better
  - Add chain file path to `geneos tls ls -l` output

## v1.9.1 Changes

* `pkg/geneos`
  - Move Netprobe XML structs to their own package `pkg/geneos/netprobe`

---

## Version v1.8.1

> **Released 2023-09-01**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.8.1 Fixes

* [`tools/geneos`](tools/geneos/README.md)
  - `unset` should not present a warning if special parameters are passed but no actions performed, e.g. removing a non-existing environment variable
  - [#181](https://github.com/ITRS-Group/cordial/issues/181) - now build on MacOS, primarily for remote admin. Not fully tested
  - [#182](https://github.com/ITRS-Group/cordial/issues/182) - a slew of issues around the order of actions during package install, uninstall and update fixed

---

## Version v1.8.0

> **Released 2023-08-16**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.8.0 Highlights

This version changes the way [`geneos tls import`](tools/geneos/docs/geneos_tls_import.md) works to add support for the import of external "real" certificates into your Geneos environment. You can now supply a PEM formatted certificate and key with an options verification chain and add these to existing instances.

Key files are now enabled for use by default for all new Gateway instances. Key files have been automatically created for some time now, but not automatically enabled in the starting environment of the Gateway.

## v1.8.0 Changes

* [`tools/geneos`](tools/geneos/README.md)
  - Enable the use of external key-files for all *new* Gateways running on version GA5.14 and above. Existing Gateways will not be affected as the default is `usekeyfile=false`. If you do not want to use an external key-file set `usekeyfile=false` before starting for the first time. If a Gateway has been started with or without a keyfile and created a cache directory then you must follow the instructions in the documentation, <https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm#How_to_change_the_key_file_of_your_Gateway>, otherwise your Gateway will not start-up.

  - New options to the `start`, `restart` and `command` sub-commands allow you to add one-off command line arguments and environment variables to an instance. This is useful, for example, to pass a `-skip-cache` argument to a Gateway.

  - Extensive rework to the internal handling of loops-over-instances to pass-back an `instance.Response` struct and handle output at the caller. This is preparation for work on non-CLI interfaces (think: REST API and web). This may break some output formatting, please report via github issues.

  - `tls import` has changed to support the import of instance certificate, signing certs and chains in a more organised way. It is unlikely anyone was using the previous incarnation which was highly limited but just in case, this is a **breaking change** to the syntax and functionality of `tls import`.

* [`pkg/geneos/api`](pkg/geneos/api/README.md)
  - A new API for inbound data to Geneos. This package is not yet ready for real-world use.

## v1.8.0 Fixes

* [`pkg/config`](pkg/config/README.md)
  - [#176](https://github.com/ITRS-Group/cordial/issues/176) fix support for Windows paths in `${enc:...}` expansion formats

* [`tools/geneos`](tools/geneos/README.md)
  - A fix for a long time bug in an internal routine that checked reserved names. This was found during the refactoring of code above. Oddly this doesn't appear to have been noticed, not sure why.

  - Fix closing of open file descriptors when starting a local instance. This needed cmd.Extrafiles slice having empty nils added through the the largest FD.

  - Fix merging of aliases during instance config load.

---

## Version v1.7.2

> **Released 2023-07-28**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.7.2 Changes

* `pkg/gwhub`, `pkg/icp`, `pkg/streams` and `pkg/geneos` have been updated to match real APIs and to add access to REST API streams

## v1.7.2 Fixes

* [#172](https://github.com/ITRS-Group/cordial/issues/172) - viper doesn't do the right thing with overridden values in maps containing defaults. This would affect GetStringMap*() callers, and we also now have our own UnmarshalKey() function

* `pkg/config` and `tools/geneos`: Fix handling of command line plaintext passwords (those not prompted for). When passed a pointer to a method you have to set the destination of the pointer, not the ephemeral pointer itself

* `tools/geneos` would not correctly initialise web server directories after changes to import earlier in v1.7. This is now fixed along with the removal of a confusing treatment of "~/" in an import path not meaning the user's home directory

---

## Version v1.7.1

> **Released 2023-07-25**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.7.1 Changes

* `pkg/gwhub` & `pkg/icp` - Updates for ongoing project

## v1.7.1 Fixes

* [#167](https://github.com/ITRS-Group/cordial/issues/167) - Only load template files with a `.gotmpl` extension.

* [#169](https://github.com/ITRS-Group/cordial/issues/169) - If the file being imported is the *same* as the destination, skip the copy.

* `tools/geneos` - Fix installation of packages from local sources with or without component on command line

---

## Version v1.7.0

> **Released 2023-07-11**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.7.0 Changes

* `tools/geneos` - Optimisation and parallel execution. All operations on Geneos instances are now run in parallel which has resulted in significant improvements in responsiveness and has reduced delays waiting for things to happen on larger installations and remote hosts. While extensive testing has taken place to ensure that the underlying functionality is not affected, there may still be issues in some cases - please report them as soon as you can!

* Add support for TLS key type selection, defaulting to ECDH (see `geneos help tls init`)

* Split `help` and `-h` options - `help` now gives the long description and usage while `--help`/`-h` only gives short description plus usage

* `pkg/geneos` updates to XML parsing structures, fix regex handling

* `pkg/gwhub` updates for better API support (work in progress)

* `pkg/config` updates, with some API changes, to better support `tools/geneos` configuration handling and other refactoring and update ExpandString option NoDecode()

* Use `upx` for compression of binaries during releases build - saves about 2/3rd space

* Make consistent the handling of TLS certs and keys internally

* `geneos ps` will show the actual version of each instance running, in case the base symlink has been updated and the process not restarted

* Quite a bit of redecorating inside `tools/geneos` internal packages to make things clearer (refactoring, merge and split of functions etc.)

* `tools/geneos` - Initial support for "remote only" working; i.e. if GENEOS_HOME is not set but there are remotes then try to "do stuff". This will break if you perform a local operation such as `add` as the root then is the current directory. Further work required, but getting Windows support working again is on the way.

* `tools/geneos` - Add a basic `--install` option to `package update` to allow checking of package that match the ones being updated and download them if found.

## v1.7.0 Fixes

* [#156](https://github.com/ITRS-Group/cordial/issues/156) - fix progressbar newline issue

* [#155](https://github.com/ITRS-Group/cordial/issues/155) - refactor instance home directory handling (mostly internal)

* [#153](https://github.com/ITRS-Group/cordial/issues/153) - fix local install of only components available

* `tools/geneos` - fix order of columns in plain `geneos ls`

* [#38](https://github.com/ITRS-Group/cordial/issues/38) - fix update stop/start as well as a number of related issues in `package install` and the handling of `fa2` packages

* [#152](https://github.com/ITRS-Group/cordial/issues/152) - call Rebuild() on *every* instance config save - then instance.setup.xml will stay in sync with config

* [#150](https://github.com/ITRS-Group/cordial/issues/150) - document `deploy` behaviour when versions clash

## 1.7.0 Known Issues

* [#165](https://github.com/ITRS-Group/cordial/issues/165) - restarting while updating SANs is not working

---

## Version v1.6.6

> **Released 2023-06-28**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.6.6 Fixes

* [#145](https://github.com/ITRS-Group/cordial/issues/145) - Wrap `geneos tls list` JSON output in an array correctly

* `tools/geneos` - Package version handling was broken in some cases, especially for components with 'parent' types

## v1.6.6 Changes

* `tools/geneos` - Add a progress bar for downloads when running interactively. Make checking if running interactively consistent

* `tools/geneos` - Add `package install -D` to download packages without unpacking them

* `tools/geneos` - Refactor internal variable names in subsystem packages to shrink very long names

* `pkg/geneos` - Various updates to plugins and other structures to support an ongoing project

---

## Version v1.6.5

> **Released 2023-06-23**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.6.5 Fixes

* [#146](https://github.com/ITRS-Group/cordial/issues/146) - Entering empty passwords could cause pointer panics

* [#148](https://github.com/ITRS-Group/cordial/issues/148) - Fallback to environment vars when user.Current() fails because user is not in local passwd file with static binary.

---

## Version v1.6.4

> **Released 2023-06-22**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

Note: v1.6.3 was removed, and v.1.6.4 releases to address some last minute issues.

## v1.6.4 Changes

* `tools/geneos` - Add an initial `hidden` flag to hosts, so they don't participate in wildcarded operations.

* `tools/geneos` - Add new `--validate` option to `geneos show` to run a validation and output results as JSON. 

* `pkg/geneos` - Updates for further parsing of config files, fix the treatment of `geneos.Value` type.

* `pkg/config` - Add a `config.UseEnvs()` option to `New()` and `Load()` to trigger viper `AutomaticEnv()` and use prefixes.

* Convert all packages and programs to use `*config.Plaintext` and not `config.Plaintext`.

## v1.6.4 Fixes

* Fix remote host optional encoded password handling

* [#142](https://github.com/ITRS-Group/cordial/issues/142) - Fix expansion of non-encoded config strings in `show` and other places

* [#140](https://github.com/ITRS-Group/cordial/issues/140) - Fix generation of 'secure' args for command start-up

* [#138](https://github.com/ITRS-Group/cordial/issues/138) - Fix autostart behaviour for `geneos restart`

* [#139](https://github.com/ITRS-Group/cordial/issues/139) - Show running AC2 instances

* [#134](https://github.com/ITRS-Group/cordial/issues/134) - Update some `geneos` commands that need either flags or args set to just output usage otherwise.

* [#133](https://github.com/ITRS-Group/cordial/issues/133) - Check restart logic and fix for when instance is already stopped. Also update the Stop() function and it's usage in other callers.

---

## Version v1.6.2

> **Released 2023-06-14**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.6.2 Fixes

* `tools/geneos` Fix late found bug with `deploy` and home directories

---

## Version v1.6.1

> **Released 2023-06-13**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.6.1 Changes

* `pkg/geneos` Changes

  Some API changes to support work on a project for reading Gateway configuration files. Existing structures used to write config files cannot co-exist and have been renamed with an "Out" suffix down to the SamplerOut level. While the old names should have been retained and the new API requirements used new names, it was decided that this is the more common use case in the future.

* `tools/geneos` Command updates

  The `show` command can now output an instance's own configuration file (for types of Netprobe and Gateway) and also try to produce a merged Gateway file using a modified command line with the Gateway `-dump-xml` command line option.

  A new instance flag `autostart` has been added, set to `true` for all types except `ac2` which defaults to `false`. Documentation updtes to follow.

  The `init demo` command now detects if the user has a `DISPLAY` environment variable set and if so also installs an `ac2` instance.

  The `command` command can now output the details in JSON format. This format is not quite compatible with the `pkg/process` Run and Batch functions, but the aim is to eventually merge the formats so that they can also share the implementation later.

## v1.6.1 Fixes

* Minor ongoing changes

  While adding new features there is ongoing review work and refactoring of code.

---

## Version v1.6.0

> **Released 2023-06-07**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.6.0 Changes

* [#116](https://github.com/ITRS-Group/cordial/issues/116)

  Added a new [`geneos deploy`](tools/geneos/docs/geneos_deploy.md)
  command that combines `geneos init` and `geneos add` but takes more
  care over existing installations and/or creating new directories.

  As part of this work all the `geneos init` command will prompt the
  user for a directory if none is given on the command line. If the
  command is run from a non-interactive parent (e.g. a pipe from the
  shell) then the prompt is skipped and the default directory is used.
  
  The `geneos deploy` command uses the same initialisation rules but
  reduces the number of options. The intended audience is around
  automation where the deployment scripts may not have the knowledge or
  logic to check for existing installations.

* [#114](https://github.com/ITRS-Group/cordial/issues/114)

  For `geneos` instances that have both the default `libpaths` and an
  environment variable `LD_LIBRARY_PATH` configured these are now
  concatenated with `libpaths` always first.

* [#117](https://github.com/ITRS-Group/cordial/issues/117)

  Based on user feedback all the Netprobe types have been merged under
  the `netprobe/` directory in their respective plural names, e.g.
  `netprobe/sans`. Existing installations should continue to work
  unchanged but you can use the `geneos migrate` command to
  automatically merge the instance directories under `netprobe/`
  including the update of configuration files.

* [#97](https://github.com/ITRS-Group/cordial/issues/97)

  The Linux Active Console is now treated like any other component and
  instance. At the moment, if you issue a `geneos start` command then
  all instances including Active Console(s) will be run. In a future
  release we may add an `autostart` like flag that can prevent this and
  require a manual start using the full `geneos start ac2 abcde` syntax.

* [`pkg/process`](pkg/process/) - New features

  New functions have been added to support the running of single
  processes and batches based on a Program struct. This is for running
  tasks loaded from a config file (typically YAML) for an ongoing
  project. The reason for not using existing external packages was the
  integration with other `cordial` tooling. This functionality is
  currently maturing and is very sparsely documented and subject to
  major changes.

* [`pkg/icp`](pkg/icp) and [`pkg/gwhub`](pkg/gwhub) - New APIs

  These two packages are the start of Go APIs for ITRS Capacity Planner
  and Gateway Hub, respectively. These should not yet be used and have
  been included to track progress over the next few releases.

## v1.6.0 Fixes

* [#126](https://github.com/ITRS-Group/cordial/issues/126)

  In the [config package](pkg/config/) the Load() function would fail if
  used with a file format set bu other defaults and run in the same
  directory as the binary it ran in because viper would also try to load
  the bare-named program binary as a config file of the type given. The
  package now does it's own file name construction to avoid this.

  As a consequence of the work done around this fix to make the usage of
  options to Load() and Save() clearer some have changed names. Existing
  code that wants to use v1.6.0 will experience minor API breakage. The
  fixes are simple refactors, so no backward compatibility has been
  retained.

* [#124](https://github.com/ITRS-Group/cordial/issues/124)

  The work done for
  [#117](https://github.com/ITRS-Group/cordial/issues/117) above meant
  that all templates are now located under `netprobe/templates` and both
  `san` and `floating` templates had the same name. The default
  templates now have updated root names, e.g. `san.setup.xml.gotmpl`,
  but existing configuration may need updating if the existing templates
  clash.

  To help users control which configuration files are created from
  templates, and which to use for instance start-up, a new instance
  parameter `setup` has been introduced for this. The defaults are
  `netprobe.setup.xml` and `gateway.setup.xml` for the two affected
  component types which means no change for existing users.

  So even though the new SAN template is `san.setup.xml.gotmpl`, for
  example, running `geneos rebuild san` will still result in a
  `netprobe.setup.xml` in the instance directory.

* [#120](https://github.com/ITRS-Group/cordial/issues/120)

  While this report was a misunderstanding of the way to use Daemon()
  the comments have been updated to give better direction on how to use
  the pidfile io.Writer parameters.

---

## Version v1.5.2

> **Released 2023/05/31**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.5.2 Changes

  * [#102](https://github.com/ITRS-Group/cordial/issues/102) - `process`
    package additions

  * [#109](https://github.com/ITRS-Group/cordial/issues/109) - New `tls
    create` command

    Move the functions of `--name` etc from `tls new` to `tls create` to
    remove dependency on the Geneos home directory.

  * [#106](https://github.com/ITRS-Group/cordial/issues/106) - Change
    directory for TLS root and signing certs and keys

  * [#97](https://github.com/ITRS-Group/cordial/issues/97) - Start of
    Linux AC2 support. Not yet fully functional.

  * [#98](https://github.com/ITRS-Group/cordial/issues/98) - Work done
    and then superceeded by
    [#109](https://github.com/ITRS-Group/cordial/issues/109) above.


## v1.5.2 Fixes

  * [#111](https://github.com/ITRS-Group/cordial/issues/111) - Fix
    gateway instance template ports

  * [#103](https://github.com/ITRS-Group/cordial/issues/103) - Fix
    default web server log file name

## Version v1.5.1

> **Releases 2023/05/25**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.5.1 Changes

* [`tools/geneos`](tools/geneos/README.md):

  * [#85](https://github.com/ITRS-Group/cordial/issues/85) - TLS Issues

    Added verification for certificates against existing root and
    signing certificates.

    Added an option to recreate root and signing certs from `tls init`.

  * [#86](https://github.com/ITRS-Group/cordial/issues/86) - New log
    option for stderr logs blocked main logs

    Reworked the way that logs are followed to allow both normal and
    stderr logs to be followed. This fixes `start -l`, `restart -l` and
    by implication `init -l` options. Added a `--nostandard`/`-N` option
    to `logs` to allow normal log to be surpressed when you only want
    the error logs.

  * [#88](https://github.com/ITRS-Group/cordial/issues/88) - Floating
    probe configuration template output malformed

    The defaults for new floating probes used `sanname` instead of
    `floatingname` as the parameter for the template rendering.

  * [#90](https://github.com/ITRS-Group/cordial/issues/90) - Variables
    in SANs

    The san template was not corrected updated for the new variable
    structure.

  * [#43](https://github.com/ITRS-Group/cordial/issues/43) - Minor error
    in `disable`
   
    In `disable` check if stopped and print error if not `--force`

---

## Version v1.5.0

> **Released 2023/05/22**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

### v1.5.0 - Highlights

This release brings numerous changes to the `cordial` tools, especially
the `geneos` utility. We have tried to improve the reliability and
usability of the utility through updated and improved documentation and
subsequent fixes and changes that arose from writing and checking that
documentation.

### v1.5.0 - **BREAKING CHANGES**

  * `pkg/logger`:

    * **REMOVED**. This simplistic internal logging package has been
      deprecated in favour of more mature solutions, such as
      [zerolog](https://pkg.go.dev/github.com/rs/zerolog).

  * `tools/geneos`:

    * The way SAN instances handle gateway connections has been fixed to
      allow gateway represented as FDQNs or IP addresses. The old way
      resulted in a mess - viper defaults to dots ('.') as configuration
      item hierarchy delimiters and this caused issues. Most users with
      SANs should not notice any change, but if you see problems please
      check the san XML file and correct the `gateways` section as
      necessary. One way is to remove and re-set them using:
      
      > `geneos set san mySan -g gateway.example.com ...`

      Running `set` will rewrite the configuration in the new format bu
      there is a chance that the previous configuration will continue to
      occupy settings. You may need to manually edit the instance
      configuration file `san.json` anyway.

    * Like the above any variables defined for either SAN or Gateway
      instances used to generate XML from templates will have not worked
      in a case sensitive manner to mirror how Geneos treats variable
      names. To fix this the internal format of the `variables` section
      has been updated to move the variable name from the configuration
      key to a separate structure as it's own value. Code has been added
      to automatically convert from the old format to the new when the
      configuration file is updated however there is no fix for the
      correction of variable case name being incorrect from previous
      configurations. Please review and adjust as necessary.

    * Support for running instances as other user accounts or under
      `sudo` has been deprecated. Security is _hard_, and the support
      for these was poorly implemented. A better way should be coming in
      a future release.

      This may mean that where users has configured netprobes to run as
      different users and have previously run `sudo geneos start` to let
      the program do the right thing will run into issues. Please be
      careful if any of your instances run as other users and do not run
      the `geneos` program with `sudo`. There is no additional
      checking/rejection of running under `sudo` or any other privilege
      escalation system so this is important!

### v1.5.0 - Other Changes

  * There has been a significant amount of refactoring and moving around
    of the code-base. Most of this should not be user visible, but some
    public APIs have changed. As with all major changes there may be
    problems that have not been caught in testing. Please report
    anything you see as either a [github
    issue](https://github.com/ITRS-Group/cordial/issues) or via the
    [ITRS Community Forum](https://community.itrsgroup.com/).

    There are too many changed to list them all in detail but specific
    ones worth mentioning include:

    * [`memguard`](https://pkg.go.dev/github.com/awnumar/memguard)
      support for protected memory. Credentials (passwords, TLS keys and
      so on) should now be handled as Enclaves (for plaintext or private
      keys) or as LockedBuffers (for ciphertexts of sensitive data).

      The [`config`](pkg/config/README.md) package includes new methods
      for handling configuration file data as Enclaves and LockedBuffers
      to try to reduce the amount of confidential data visible in the
      process.

      The changes are ongoing and, in addition to adding a layer of data
      security to `cordial`, an added benefit is the interception of
      memory use errors etc. If you see errors, panic etc. please report
      them as a [github
      issue](https://github.com/ITRS-Group/cordial/issues)

    * A number of the previous package APIs have undergone review and
      changed as needed. In particular the
      [`config`](pkg/config/README.md) API has been through the wringer
      and if you have any code that relies on it from v1.4 or earlier
      then it will require changes. There are new functions, which is to
      be expected, but also some existing ones have been renamed or had
      their argument signatures changed. Please review the documentation
      to see what the methods and functions have become.

    * Credentials support. There is both general purpose and
      [`geneos`](tools/geneos/README.md) specific support for the local
      storage of credentials. Passwords and other secrets "at rest" are
      stored in Geneos AES256 format using a key file that is initial
      auto-generated. To decode these passwords you must have both the
      key file (which is by default only user readable) and the
      credentials file. There should be support for other credentials
      types, such as OAuth style client secrets and tokens, in future
      releases. The _username_ and the _domain_ that the credentials
      apply to are not encrypted, by design. This is however subject to
      change in a future release.

      Credentials currently works with a free-text domain that matches a
      destination using a "longest match wins" search, e.g. for a URL
      this may be a full or partial domain name, and for Geneos
      component authentication, e.g. the REST command API, the domain is
      in the form `gateway:NAME`. Others will be added later, probably
      including TLS certificates and keys as well as SSH password and
      private keys.

    * Releases now include selected binaries with a semantic version
      suffix. The programs in `cordial` use the base name of the binary
      as a key to select which configuration files to load, so that
      renaming the binary will result in a different set of
      configuration file being used, automatically.

      To make life simpler, any version suffix is automatically stripped
      if, and only if, it matches the one used to build the binary. This
      means you can now download `geneos-v.1.5.0` and use it without
      having to rename it (useful for initial testing of new releases).

  * [`tools/geneos`](tools/geneos/README.md):

    * Extensive documentation restructuring and rewriting. This is still
      work in progress but largely complet. Built-in help text (shown
      with the `help` command or the `--help`/`-h` option) should now
      align much more closely with real functionality and the online
      documentation is now almost completely built from the same source.

    * Addition of _subsystems_ to group commands.

    * Move `aes` and `tls` command sources to their subsystems.

    * Add `host` and `package` subsystems and create aliases for the
      original commands, e.g.
      * `add host` becomes `host add`
      * `install` becomes `package install`
      * etc.

    * The `set user`, `show user` etc. commands are now under single
      `config` subystem, e.g. `geneos config set mykey=value`

    * The `set global` and related commands have been deprecated.

    * The new `package` subsystem command pulls all Geneos release
      management into one place

    * New `login` and `logout` commands to manage credentials.

    * New `ca3` and `floating` components for Collection Agent 3 and Floating
      Netprobes

  * [`tools/dv2email`](tools/dv2email/README.md):

    * This new utility can be run as a Geneos Action or Effect to
      capture one or more Dataviews and send as an email. The
      configuration is extensive and the layout and contents are
      completely configurable through the use of Go templates.

### v1.5.0 - Bug Fixes

  * [`tools/geneos`](tools/geneos/README.md):

    * Version checking of local release archives was broken because of
      overloading of a common function. This is now split and checking
      should work once again.

    * Most reported issues on github have been fixed.

### v1.5.0 - To Do

  * Documentation needs more work and refinement. The built-in help for
    almost all commands is now up-to-date but the `init` and `tls`
    subsystems need to be reviewed further and completed. This should be
    in a patch release soon.

  * [`tools/geneos`](tools/geneos/README.md):

    * Local storage of encrypted passwords for remote SSH access needs
      documenting

---

## Version v1.4.4 - 2023/04/12

* Fixes

  * New `Default` expand option should NOT itself default to `nil`

---

## Version v1.4.3 - 2023/04/12

* Fixes

  * tools/geneos: fix `ps` not showing open ports on systems with IPv6 enabled
  * tools/geneos: make `ls` and `ps` command flags more consistent
  * tools/geneos: add an -el8 runtime to docker images when built
  * tools/geneos: fix RHEL8/Centos8 download support for localhost using new SetStringMapString() method
  * pkg/config: add SetStringMapString() methods to support settings maps (which viper doesn't support until you write a file out and read it back)
  * tools/geneos: adjust the way we choose package version, convert "-el8" to "+el8" to satisfy semver ordering
  * tools/geneos: package version number are now prefixes only

* Changes

  * tools/geneos: add `-f` flag to `ps` command to show open files. formatting subject to change for now.
  * tools/geneos: add a `update ls` command to show available package versions
  * pkg/config: added more ExpandOptions and support more Get* functions such as GetInt
  * pkg/geneos: added more Geneos XML config support, specifically Sampler Schemas and Standardised Formatting
  * libraries/libemail: added initial msTeams notification function

---

## Version v1.4.2 - 2022/12/21

* Fixes

  * tools/geneos: fix `update` to only optional restart (`-R`) the component type given
  * tools/geneos: check RHEL8 download in a case independent way - fixes remotes
  * tools/geneos: create user config directory for remote hosts in case of old location for main config
  * tools/geneos: `install` should error out is passed `@host` instead of `-H host`
  * tools/geneos: ssh known hosts handling improved (for mixed IP / hostnames)
  * tools/geneos: remote hosts with IP names are now renamed `A-B-C-D` to avoid issues with viper names

---

## Version v1.4.1 - 2022/12/19

* Fixes

  * tools/geneos: check return from user.Current() as it can fail (but shouldn't)
  * tools/geneos: numerous fixes for logic around handling of remote hosts
  * tools/geneos: fix remote host naming to be work with capitalisations
  * tools/geneos: actually load SSH private key files, if available
  * tools/geneos: re-order SSH HostKeyAlgorithms so that, bizarrely, IP based remotes work
  * tools/geneos: better handling of instance config aliases when writing config files
  * tools/geneos: fixes to unset to ignore values that may be passed in with keys to unset
  * tools/geneos: refactor CopyInstance() to preserve ports, other details
  * build: create static executables, using alpine, and a centos 7 compatible libemail.so
  * tools/geneos: add the beginnings of support for YAML instance config files. not enabled yet.
  * tools/geneos: fix crash when importing to common directories of components without the component name
  * tools/geneos: fix fileagent support by adding implicit imports with side-effects for all component packages
  * tools/geneos: skip failed permissions on /proc/*/fd - let 'ps' work for restricted processes
  * tools/geneos: fix update-during-install support, add --force flag for this too
  * tools/geneos: fix logic to match latest packages when major number changes

* Changes

  * tools/geneos: clean-up various comments, refactor methods, add license/copyright notices to many files
  * pkg/config: Add an options `expr` prefix to expansion items which supports <https://pkg.go.dev/github.com/maja42/goval> syntax
  * pkg/config: API change: Add options to the config expansion functions rather than just lookup maps
  * tools/geneos: add SSH password support for remote hosts
  * tools/geneos: support embedded SSH passwords in hosts config, using new 'set host' sub-command
  * tools/geneos: support additional SSH private key files per host via 'set host sshkeys=X,Y' sub-command
  * tools/geneos: begin implementation of support for YAML config files via 'configtype' user setting
  * pkg/geneos: add EnvironmentRef and fix periodStartTime attribute

* Other

  * tools/geneos: ongoing documentation and command help usage updates
  * tools/geneos: update README.md with more information about instance configuration files and their values (@gvastel)

---

## Version v1.3.2 - 2022/11/02

* Fixes

  * tools/geneos: fix running as root (or via sudo) and creation of config directories and file ownerships
  * tools/geneos: fix creation of full user config directories when running 'set user'

---

## Version v1.3.1 - 2022/11/01

* Fixes

  * tools/geneos: chown files and directories creates when run as root
  * tools/geneos: ensure plain 'init' creates all components dirs

---

## Version v1.3.0 - 2022/10/25

* Changes

  * PagerDuty integration
  * Merged ServiceNow integration, single binary build
  * tools/geneos: add instance protection against stop (and related) or delete commands
  * tools/geneos: support legacy command through emulating `*ctl` named commands
  * tools/geneos: allow remote operations without local directories

* Fixes

  * tools/geneos: fix logic around creating user default AES keyfiles and directory permissions
  * tools/geneos: round certificate expiry to midnight
  * tools/geneos: round tls remaining column to seconds correctly
  * tools/geneos: fix webserver command build typo. now webserver starts correctly
  * libemail: fix default _SUBJECT handling
  * tools/geneos: split over complex 'init' command into sub-commands
  * updated command usage information and reordered various internal function calls
  * tools/geneos: add password verify to aes encode and a --once flag to override
  * tools/geneos: add local JoinSlash and Dir to use Linux paths on Windows builds
  * tools/geneos: fix ssh-agent support on windows
  * tools/geneos: build on windows
  * integrations: Add PagerDuty integration
  * Integrations: Merge ServiceNow binaries into one
  * tools/geneos: change internal remote Stat() API

---

## Version v1.2.1 - 2022/10/11

Final release after numerous small fixes.

---

## Version v1.2.1-rc3 - 2022/10/07

* Fixes

  * `geneos` command fixes:
    * Fixed `init` download credential handling
    * Fixes JSON output format from `ls` commands
    * Local-only installs now work again (including default "latest" support)

  * Security
    * Updated Labstack Echo to 4.9.0 to address security advisory
      [CVE-2022-40083](https://nvd.nist.gov/vuln/detail/CVE-2022-40083).
      To best of our knowledge this particular set of features was never
      used in this package.

* Additional features and improvements

  * `geneos` command improvements:
    * Added `--raw` to `show` to not output expanded configuration values
    * Many improvements and changes to the new `aes` sub-commands. Please see [documentation](tools/geneos/README.md) for details
    * Removed built-in opaquing of credentials in output in favour of new `${enc:...}` support

  * `libemail.so` gets direct passwords back, with ExpandString support. See [documentation](libraries/libemail/README.md) for details

  * General package improvements
    * Enhanced `OpenLocalFileOrURL` to support `~/` paths
    * Enhanced `ExpandString` to support direct file paths and updates package docs further

---

## Version v1.2.1-rc1 - 2022/09/28

* Fixes

  * `geneos` instance configuration files now have expansion applied to
    string-like values. This means, for example, that changing the
    `version` of an instance from `active_prod` will correctly be
    reflected in the executable path and library paths. Previously these
    needed to be manually changed. Please note that existing instance
    configuration files will NOT be updated and will require editing.
    You can go from:

        "program": ".../packages/gateway/active_prod/gateway2.linux_64",

    to

        "program": "${config:install}/${config:version}/${config:binary}",

  For a complete list of supported expansions see `ExpandString()` in the [`config`](pkg/config/README.md) package.

* Additional features and improvements

  * `ExpandString()` was enhanced to add a `config:` prefix so that
    configurations with a flat structure, i.e. no "." in names, could be
    referenced.
  * To support the changes above in instance configurations a new method
    was added - `ExpandAllSettings()` - and the `geneos show` command
    enhanced to display both expanded and raw configurations via the new
    `--raw` flag.
  * Additional configuration item support in the
    [`geneos`](pkg/geneos/README.md) package

---

## Version v1.2.0-rc2 - 2022/09/26

* Fixes found during testing

  * Removed support for `$var` format expansion, now it's `${var}` only.
    This prevents configuration issues when, for example, plain text
    passwords contain dollar signs. The documented workaround if you
    need to include literal `${` in a configuration value still applies.

* Additional features and improvements

  * Added command `geneos aes update`. This may still be renamed before final release to `geneos aes import` depending on feedback.
  * Improvements to `geneos aes new`
  * Improvements, clarification to package and function documentation
  * Code clean-up and refactor to make some internals more understandable and to remove code duplication

---

## Version v1.2.0-rc1 - 2022/09/21

* Breaking Changes

  There are quite a lot of changes to the various components and
  packages since the original v1.0.0. Given that almost no-one outside
  the components contained in the repo itself is using the public
  package APIs I have broken the rules around semantic versioning and
  changed parts of the API.

* Highlights

  * Package changes
    * **Breaking changes**: Geneos `api` and `api-streams` XML-RPC supporting packages have had a big clean-up to make them easier to use
    * New `config` package to overlay `viper` with support for value expansion and crypto convenience functions
    * New `geneos` package to aid in the construction of XML configurations for Gateway and Netprobe. This is work in progress.
    * New `commands` package to provide the start of support for REST API Commands to the Gateway. This is work in progress.
    * New `xpath` package to work with the above and also the base for the `snapshot` command below. This is also work in progress.
    * New `cordial` package that initially carries a version constant.
    * New `process` package, providing a way to Daemon()ise callers on both Linux and Windows.
    * **Deprecation Notice**: The `logger` package will probably be removed as it was a stop-gap and is slowly being replaced with `zerolog`
  * Addition of the following commands to `tools/geneos`:
    * `aes` - Manage Geneos key-files and encoding/decoding of values
    * `snapshot` - Take dataview snapshots directly from the command line (requires GA5.14+)
  * ServiceNow integration updates
    * Configuration support is now direct with `config` above, allowing full value expansions support, including encoded credentials.
  * Logging changes
    * The logging in `tools/geneos` has been migrated to `zerolog` from the internal `logger` for a more flexible package. This will be further rolled-out to other parts of the repo in time.

---

## Version v1.0.0 - 2022/06/14

* First Release
