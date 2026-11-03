# Cordial

Cordial provides tools and integrations to help you and your team manage ITRS Geneos. It makes routine work easier to repeat, with ready-made commands, reports, and integrations that reduce manual steps and the number of scripts you need to maintain. Geneos can monitor many Gateways, Netprobes, hosts, and plugins. To see how these components collect, process, and present monitoring data, read the [Geneos architecture guide](https://docs.itrsgroup.com/docs/geneos/current/getting-started/architecture/index.html).

Cordial does not replace any existing Geneos components. It helps you get more operational value from it with less manual effort and fewer tools to maintain. Depending on how you use it, Cordial can also reduce the effort and cost of routine support.

## Operational workflows

### Manage Geneos with one command

The [`geneos` command](tools/geneos/README.md) gives you one consistent way to view, install, update, configure, start, and stop Geneos components. You can manage local and remote hosts, work with existing installations, and use profiles as starting points for common deployments. This reduces repeated host-by-host work and makes routine changes easier to script and repeat. Profile and Monitor-of-the-Monitors features in the current development release are previews.

### Review monitoring coverage with GDNA

Geneos licence data can be detailed and spread across many Gateways and Netprobes. [GDNA](gdna/README.md) turns data from Geneos Licence Daemons (`licd`) into dashboards and reports that show monitoring coverage across servers; it does not collect monitoring data from other sources. You and your team can use these reports to find potential gaps, compare plugin use with your coverage goals, and plan improvements. GDNA can also email reports on a schedule, so you can share regular updates without running them manually. Results depend on the licence data available and the coverage policies in use. To learn about the data source, read the ITRS documentation for the [Geneos Licence Daemon](https://docs.itrsgroup.com/docs/geneos/7.12.0/administration/licence-daemon/index.html).

### Share Dataviews with dv2email

[`dv2email`](tools/dv2email/README.md) turns Geneos Dataview snapshots into email reports or files such as HTML and Excel workbooks. A Gateway Action or Effect can start a report when an alert occurs. This avoids repeated manual exports and copy-and-paste for incident reviews, handovers, and other operational reports. To see how the Gateway provides Dataview snapshots, read the [Gateway Commands guide](https://docs.itrsgroup.com/docs/geneos/current/processing/monitoring-and-alerts/geneos_commands_tr/index.html#rest-service).

### Connect alerts to ServiceNow

The [`servicenow2` integration](integrations/servicenow2/README.md) maps Geneos alert data to ServiceNow incident fields. Based on its configured rules and correlation values, it can create an incident or update an existing one. It removes the need for a separate wrapper script to translate Gateway event data. This reduces manual ticket entry and keeps alerts in your service-management workflow. To see how Geneos passes event details to external programs, read the ITRS guide to [Gateway Actions and Effects](https://docs.itrsgroup.com/docs/geneos/current/processing/monitoring-and-alerts/geneos_rulesactionsalerts_tr/index.html).

## Low-overhead deployment

The pre-built Linux command-line programs are statically linked for compatible 64-bit x86 Linux systems. You do not need to install a separate language runtime, interpreter, or helper utility to run them. The exception is `libemail.so`: it is a shared library that uses the host's standard C runtime and must be loaded by a compatible Geneos Gateway. Each program still needs the configuration, network access, and external services required for its work.

The `geneos` command also includes tools for many tasks that might otherwise need separate scripts or diagnostic utilities. For example, `geneos package install` can download and unpack releases; `geneos ps` can inspect running instances, open files, and sockets; `geneos logs` can read, filter, and follow local or remote logs; and `geneos snapshot` can fetch Gateway Dataviews. These built-in tools can reduce software setup and maintenance on your Geneos hosts.

## Integrations

| Integration | Purpose |
| --- | --- |
| [`servicenow2`](integrations/servicenow2/README.md) | Creates or updates ServiceNow incidents from Geneos alerts. |
| [`pagerduty`](integrations/pagerduty/README.md) | Sends Geneos alerts to PagerDuty as events. |

## Libraries

| Library | Purpose |
| --- | --- |
| [`libalert`](libraries/libalert/README.md) | Geneos notification library with enhanced email support and Microsoft Teams notifications. |
| [`libemail`](libraries/libemail/README.md) | Geneos email library with support for authenticated SMTP, TLS, and templates. |

## Other tools

| Tool | Purpose |
| --- | --- |
| [`files2dv`](tools/files2dv/README.md) | Builds Geneos Toolkit-compatible Dataview data from files. Its background push mode is not implemented. |
| [`ims-gateway`](tools/ims-gateway/README.md) | Connects Geneos incident commands to ServiceNow or ServiceDesk Plus. |
| [`san-config`](tools/san-config/README.md) | Serves inventory-based configuration to Self-Announcing Netprobes. |

## Go packages

If you build your own tools, Cordial also provides Go packages that work with Geneos:

| Package | Purpose |
| --- | --- |
| [`config`](pkg/config/README.md) | Load configuration and expand values such as environment variables and encrypted credentials. |
| [`geneos`](pkg/geneos/README.md) | Build Geneos Gateway and Netprobe configuration using Go data structures. |
| [`geneos/api`](pkg/geneos/api/README.md) | Send data to Netprobes through their REST and XML-RPC APIs. |
| [`geneos/commands`](pkg/geneos/commands/README.md) | Run Gateway REST commands and retrieve Dataview snapshots. |
| [`geneos/plugins`](pkg/geneos/plugins/README.md), [`samplers`](pkg/geneos/samplers/README.md), [`streams`](pkg/geneos/streams/README.md), [`xmlrpc`](pkg/geneos/xmlrpc/README.md) | Build integrations with the Geneos XML-RPC API. |
| [`geneos/xpath`](pkg/geneos/xpath/README.md) | Work with XPath expressions used by Geneos configuration and APIs. |
| [`host`](pkg/host/README.md) | Perform file and process operations on local and remote SSH hosts. |
| [`email`](pkg/email/README.md) | Send Geneos-aware email through SMTP, including TLS and authentication. |

Other packages support certificates, logging, reporting, REST clients, process management, IMS, Capacity Planner, and Gateway Hub. For the full package list, see the [package overview](pkg/README.md).
