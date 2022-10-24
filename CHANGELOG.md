# Change Log

## Version v1.3.0-alpha - 2022/10/22

* tools/geneos: support legacy command through emulating `*ctl` named commands
* tools/geneos: allow remote operations without local directories
* libemail: fix default _SUBJECT handling
* tools/geneos: split over complex 'init' command into sub-commands
* updated command usage information and reordered various internal function calls
* tools/geneos: add password verify to aes encode and a --once flag to override
* tools/geneos: add local JoinSlash and Dir to use Linux paths on Windows builds
* tools/geneos: fix ssh-agent support on windows
* tools/geneos: build on windows
* integrations: Add PagerDuty integration
* integrations: Merge ServiceNow binaries into one
* tools/geneos: change internal remote Stat() API

## Changes

* PagerDuty integration
* Merged ServiceNow integration, single binary build

## Version v1.2.1 - 2022/10/11

Final release after numerous small fixes.

## Version v1.2.1-rc3 - 2022/10/07

### Fixes

* `geneos` command fixes:
  * Fixed `init` download credential handling
  * Fixes JSON output format from `ls` commands
  * Local-only installs now work again (including default "latest" support)

* Security
  * Updated Labstack Echo to 4.9.0 to address security advisory [CVE-2022-40083](https://nvd.nist.gov/vuln/detail/CVE-2022-40083). To best of our knowledge this particular set of features was never used in this package.

### Additional features and improvements

* `geneos` command improvements:
  * Added `--raw` to `show` to not output expanded configuration values
  * Many improvements and changes to the new `aes` sub-commands. Please see [documentation](tools/geneos/README.md) for details
  * Removed built-in opaquing of credentials in output in favour of new `${enc:...}` support

* `libemail.so` gets direct passwords back, with ExpandString support. See [documentation](libraries/libemail/README.md) for details

* General package improvements
  * Enhanced `OpenLocalFileOrURL` to support `~/` paths
  * Enhanced `ExpandString` to support direct file paths and updates package docs further

## Version v1.2.1-rc1 - 2022/09/28

### Fixes

* `geneos` instance configuration files now have expansion applied to string-like values. This means, for example, that changing the `version` of an instance from `active_prod` will correctly be reflected in the executable path and library paths. Previously these needed to be manually changed. Please note that existing instance configuration files will NOT be updated and will require editing. You can go from:

      "program": ".../packages/gateway/active_prod/gateway2.linux_64",

  to

      "program": "${config:install}/${config:version}/${config:binary}",

For a complete list of supported expansions see `ExpandString()` in the [`config`](../../pkg/config) package.

### Additional features and improvements

* `ExpandString()` was enhanced to add a `config:` prefix so that configurations with a flat structure, i.e. no "." in names, could be referenced.
* To support the changes above in instance configurations a new method was added - `ExpandAllSettings()` - and the `geneos show` command enhanced to display both expanded and raw configurations via the new `--raw` flag.
* Additional configuration item support in the [`geneos`](../../pkg/geneos) package

## Version v1.2.0-rc2 - 2022/09/26

### Fixes found during testing

* Removed support for `$var` format expansion, now it's `${var}` only. This prevents configuration issues when, for example, plain text passwords contain dollar signs. The documented workaround if you need to include literal `${` in a configuration value still applies.

### Additional features and improvements

* Added command `geneos aes update`. This may still be renamed before final release to `geneos aes import` depending on feedback.
* Improvements to `geneos aes new`
* Improvements, clarification to package and function documentation
* Code clean-up and refactor to make some internals more understandable and to remove code duplication

## Version v1.2.0-rc1 - 2022/09/21

### Breaking Changes

There are quite a lot of changes to the various components and packages since the original v1.0.0. Given that almost no-one outside the components contained in the repo itself is using the public package APIs I have broken the rules around semantic versioning and changed parts of the API.

### Highlights

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

## Version v1.0.0 - 2022/06/14

* First Release
