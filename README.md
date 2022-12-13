# cordial

* Current Version: v1.4.1-rc1 - 2022/12/12

Cordial comprises a set of interrelated programs, integrations, libraries and packages for Geneos, principally written in Go.

* [`geneos` Manager Program](tools/geneos/)
* [ServiceNow Incident Integration](integrations/servicenow/)
* [PagerDuty Integration](integrations/pagerduty/)
* [Enhanced `libemail`](libraries/libemail/)
* Go Packages
  * `commands`
    Geneos Gateway REST API Commands including programmatic support for `snapshots` of dataviews
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

See [`CHANGELOG.md`](CHANGELOG.md) for more
