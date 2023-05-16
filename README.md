# cordial

* Current Version: v1.5.0-beta - 2023/05/16

Cordial comprises a set of interrelated programs, integrations, libraries and packages for Geneos, principally written in Go.

* [`geneos` Manager Program](tools/geneos/)

* [`dv2email` Dataview to EMail](tools/dv2email)

* [ServiceNow Incident Integration](integrations/servicenow/)

* [PagerDuty Integration](integrations/pagerduty/)

* [Enhanced `libemail`](libraries/libemail/)

* Go Packages

  * `commands`

    Geneos Gateway REST API Commands including programmatic support for
    `snapshots` of dataviews

  * `config`

    Configuration file support, based on `viper` with local extensions

  * `email`

    Functions pulled from libemail to be more generally available,
    initially for the `dv2email` program.

  * `geneos`

    Automated Geneos XML configuration file generations based on Go data
    structures

  * `host`

    Remote host integration pulled from `geneos` internal packages and
    turned into an extensible interface that supports local OS and
    remote SSH/SFTP operations. This is a rough and ready API and could
    do with more review and structure.

  * `plugins`, `samplers`, `streams`, `xmlrpc`

    Geneos API plugin XML-RPC support

  * `process`

    Process management utilities.

  * `xpath`

    Geneos XPath handling

## ChangeLog

See [`CHANGELOG.md`](CHANGELOG.md) for more
