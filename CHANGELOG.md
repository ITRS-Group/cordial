# Change Log

## Version v1.4.3-dev - 2022/12/22

* Fixes

  * tools/geneos: add an -el8 runtime to docker images when built
  * tools/geneos: fix RHEL8/Centos8 download support for localhost using new SetStringMapString() method
  * pkg/config: add SetStringMapString() methods to support settings maps (which viper doesn't support until you write a file out and read it back)
  * tools/geneos: adjust the way we choose package version, convert "-el8" to "+el8" to satisfy semver ordering
  * tools/geneos: package version number are now prefixes only

## Version v1.4.2 - 2022/12/21

* Fixes

  * tools/geneos: fix `update` to only optional restart (`-R`) the component type given
  * tools/geneos: check RHEL8 download in a case independent way - fixes remotes
  * tools/geneos: create user config directory for remote hosts in case of old location for main config
  * tools/geneos: `install` should error out is passed `@host` instead of `-H host`
  * tools/geneos: ssh known hosts handling improved (for mixed IP / hostnames)
  * tools/geneos: remote hosts with IP names are now renamed `A-B-C-D` to avoid issues with viper names

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
  * pkg/config: Add an options `expr` prefix to expansion items which supports [github.com/maja42/goval] syntax
  * pkg/config: API change: Add options to the config expansion functions rather than just lookup maps
  * tools/geneos: add SSH password support for remote hosts
  * tools/geneos: support embedded SSH passwords in hosts config, using new 'set host' sub-command
  * tools/geneos: support additional SSH private key files per host via 'set host sshkeys=X,Y' sub-command
  * tools/geneos: begin implementation of support for YAML config files via 'configtype' user setting
  * pkg/geneos: add EnvironmentRef and fix periodStartTime attribute

* Other

  * tools/geneos: ongoing documentation and command help usage updates
  * tools/geneos: update README.md with more information about instance configuration files and their values (@gvastel)

## Version v1.3.2 - 2022/11/02

* Fixes

  * tools/geneos: fix running as root (or via sudo) and creation of config directories and file ownerships
  * tools/geneos: fix creation of full user config directories when running 'set user'

## Version v1.3.1 - 2022/11/01

* Fixes

  * tools/geneos: chown files and directories creates when run as root
  * tools/geneos: ensure plain 'init' creates all components dirs

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

## Version v1.2.1 - 2022/10/11

Final release after numerous small fixes.

## Version v1.2.1-rc3 - 2022/10/07

* Fixes

  * `geneos` command fixes:
    * Fixed `init` download credential handling
    * Fixes JSON output format from `ls` commands
    * Local-only installs now work again (including default "latest" support)

  * Security
    * Updated Labstack Echo to 4.9.0 to address security advisory [CVE-2022-40083](https://nvd.nist.gov/vuln/detail/CVE-2022-40083). To best of our knowledge this particular set of features was never used in this package.

* Additional features and improvements

  * `geneos` command improvements:
    * Added `--raw` to `show` to not output expanded configuration values
    * Many improvements and changes to the new `aes` sub-commands. Please see [documentation](tools/geneos/README.md) for details
    * Removed built-in opaquing of credentials in output in favour of new `${enc:...}` support

  * `libemail.so` gets direct passwords back, with ExpandString support. See [documentation](libraries/libemail/README.md) for details

  * General package improvements
    * Enhanced `OpenLocalFileOrURL` to support `~/` paths
    * Enhanced `ExpandString` to support direct file paths and updates package docs further

## Version v1.2.1-rc1 - 2022/09/28

* Fixes

  * `geneos` instance configuration files now have expansion applied to string-like values. This means, for example, that changing the `version` of an instance from `active_prod` will correctly be reflected in the executable path and library paths. Previously these needed to be manually changed. Please note that existing instance configuration files will NOT be updated and will require editing. You can go from:

        "program": ".../packages/gateway/active_prod/gateway2.linux_64",

    to

        "program": "${config:install}/${config:version}/${config:binary}",

  For a complete list of supported expansions see `ExpandString()` in the [`config`](../../pkg/config) package.

* Additional features and improvements

  * `ExpandString()` was enhanced to add a `config:` prefix so that configurations with a flat structure, i.e. no "." in names, could be referenced.
  * To support the changes above in instance configurations a new method was added - `ExpandAllSettings()` - and the `geneos show` command enhanced to display both expanded and raw configurations via the new `--raw` flag.
  * Additional configuration item support in the [`geneos`](../../pkg/geneos) package

## Version v1.2.0-rc2 - 2022/09/26

* Fixes found during testing

  * Removed support for `$var` format expansion, now it's `${var}` only. This prevents configuration issues when, for example, plain text passwords contain dollar signs. The documented workaround if you need to include literal `${` in a configuration value still applies.

* Additional features and improvements

  * Added command `geneos aes update`. This may still be renamed before final release to `geneos aes import` depending on feedback.
  * Improvements to `geneos aes new`
  * Improvements, clarification to package and function documentation
  * Code clean-up and refactor to make some internals more understandable and to remove code duplication

## Version v1.2.0-rc1 - 2022/09/21

* Breaking Changes

  There are quite a lot of changes to the various components and packages since the original v1.0.0. Given that almost no-one outside the components contained in the repo itself is using the public package APIs I have broken the rules around semantic versioning and changed parts of the API.

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

## Version v1.0.0 - 2022/06/14

* First Release
