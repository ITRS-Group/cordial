# cordial

This repo contains a set of interrelated programs, integrations, libraries and packages for Geneos principally written in Go.

* [`geneos` Manager Program](tools/geneos/)
* [ServiceNow Incident Integration](integrations/servicenow/)
* [Enhanced `libemail`](libraries/libemail/)
* Go Packages
  * `commands`
    Geneos Gateway REST API Commands
  * `config`
    Configuration file support, based on `viper` with local extensions
  * `geneos`
    Automated Geneos XML configuration file generations based on Go data structures
  * `process`
    Process management utilities.
  * `plugins`, `samplers`, `streams`, `xmlrpc`
    Geneos API plugin XML-RPC support
  * `xpath`
    Geneos XPath handling

## ChangeLog

### Version v1.2.0

#### Breaking Changes

There are quite a lot of changes to the various components and packages since the original v1.0.0. Given that almost no-one outside the components contained in the repo itself is using the public package APIs I have broken the rules around semantic versioning and changed parts of the API.

#### Highlights

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

### Version v1.0.0

Initial Release
