# `cordial` - Geneos Utilities, Tools, Integrations and Packages

> [!NOTE]
>
> This is **Version v1.15.0** - Released 2024-06-26 - See [`CHANGELOG.md`](CHANGELOG.md) for more details.

> 🎁 Introducing GDNA 🎈
>
> GDNA (Geneos Dynamic Netprobe Analysis) gives an overview of the levels of Geneos monitoring coverage in your I.T. estate and prompts action to ensure that there are no gaps in the visibility of the health of your applications and systems. It does this through analysing Geneos license allocation data and applying heuristics to plugin usage information to present visual indications of coverage levels through two dashboards.
>
> For many users installing GDNA is as easy as starting up a docker container after configuring the data sources.
>
> See the [`gdna`](gdna/README.md) README for more details.


## Tools and Utilities

| Name                                                   | Description                                                         |
| ------------------------------------------------------ | ------------------------------------------------------------------- |
| [`gdna`](gdna/README.md)                               | Geneos Dynamic Netprobe Analysis                                    |
| [`geneos`](tools/geneos/README.md)                     | Manage Your Geneos environment                                      |
| [`dv2email`](tools/dv2email/README.md)                 | Send a Dataview as an EMail                                         |
| [`files2dv`](tools/files2dv/README.md)                 | Scan directories and files for information (not an FKM substitute!) |
| [`gateway-reporter`](tools/gateway-reporter/README.md) | Generate monitoring coverage reports from setup files               |
| [`libemail`](libraries/libemail/README.md)             | Drop-In Updated Replacement for `libemail`                          |

## Integrations

| Name                                            | Description                      |
| ----------------------------------------------- | -------------------------------- |
| [servicenow](integrations/servicenow/README.md) | Send Geneos Alerts to ServiceNow |
| [pagerduty](integrations/pagerduty/README.md)   | Send Geneos to PagerDuty         |

## Packages

These packages provide Go interfaces to ITRS Geneos as well as utilities to help build useful tools for working with ITRS Geneos.

| Package                                                                                                                                              | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| ---------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`config`](pkg/config/README.md)                                                                                                                     | Configuration support, based on the excellent [`viper`](https://pkg.go.dev/github.com/spf13/viper) package, with local extensions to add expansion of embedded references including AES encrypted values, interpolation of environment variables and other configuration parameters as well as reading local files and from URLs.<br><br>A one-stop-shop to load and save configurations makes initialising programs easier and simpler.<br><br>The implementation is slowly maturing but is not complete and is subject to API changes as we find better ways to do things. The options on those functions that have been extended allow fine grained control of how interpolation is performed including restricting which methods are supported and adding custom functions for interpolation/expansion. |
| [`email`](pkg/email/README.md)                                                                                                                       | Functions extracted from the `libemail` sources to be more generally available, initially for the `dv2email` program but also for additional tools in the future. `libemail` has not been changed so that it remains fully backward compatible.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| [`geneos`](pkg/geneos/README.md)                                                                                                                     | Automate Geneos XML configuration file generation using Go programs.<br><br>The Geneos schema is not, and cannot be, fully implemented at this stage as the mappings have been hand-rolled rather than any attempt as machine translation.<br><br>Automate Geneos XML configuration file generation using Go programs.<br><br>The Geneos schema is not, and cannot be, fully implemented at this stage as the mappings have been hand-rolled rather than any attempt as machine translation.                                                                                                                                                                                                                                                                                                                |
| [`geneos/api`](pkg/geneos/api/README.md)                                                                                                             | An updated API package for sending data into Geneos. This is work in progress and is not ready for real-world use. This package will provide a unified API for both XML-RPC and REST APIs, within the constraints of the features of both.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| [`host`](pkg/host/README.md)                                                                                                                         | Remote host integration extracted from `geneos` internal packages and turned into an extensible interface that supports local OS and remote SSH/SFTP operations. The API is still in flux and could do with more review and structure.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| [`plugins`](pkg/plugins/README.md)<br>[`samplers`](pkg/samplers/README.md)<br>[`streams`](pkg/streams/README.md)<br>[`xmlrpc`](pkg/xmlrpc/README.md) | These four packages provide support for the Geneos XML-RPC API Plugin.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| [`process`](pkg/process/README.md)                                                                                                                   | Process management functions. There is a `Daemon()` function to background a process and the beginnings of program and batch managers. While the `Daemon()` function is relatively stable the other methods in this package are new and liable to change as their use matures.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| [`commands`](pkg/commands/README.md)                                                                                                                 | Geneos Gateway REST API Commands including programmatic support for `snapshots` of Dataviews. When used with the `xpath` package below it provides a simple way of executing REST commands on Geneos Gateways.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| [`xpath`](pkg/xpath/README.md)                                                                                                                       | Geneos XPath handling functions and methods. This is a developing API and is not complete. Basic functionality exists to parse and manipulate simple XPaths.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| [`pkg/icp`](pkg/icp)<br>[`pkg/gwhub`](pkg/gwhub)                                                                                                     | These two packages are the start of APIs to ITRS Capacity Planner and Gateway Hub respectively. They are work in progress and should not be used for anything other than testing for the moment.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
