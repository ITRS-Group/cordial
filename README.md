# Cordial

Cordial is a practical toolkit for the day-to-day operation of ITRS Geneos. It brings common work, such as managing deployments, reviewing monitoring coverage, sharing Dataviews, and routing alerts, into ready-made commands, reports, and integrations.

Cordial works alongside your Geneos environment; it does not replace its monitoring components. Use it to make routine work more consistent and reduce one-off scripts, manual exports, and hand-offs. This can leave you and your team with more time to review monitoring results and act on them.

## Business and operational benefits

- **Reduce operational overhead:** Common administration, reporting, and alert-routing tasks are available as ready-made tools rather than requiring teams to build and maintain local scripts. This can reduce repetitive effort and allow skilled staff to spend more time on higher-value service improvements.

- **Improve consistency and control:** Repeatable commands, standard deployment options, and recorded operational actions help organisations apply the same practices across Geneos estates. This supports more predictable delivery, clearer accountability, and easier onboarding of operational staff.

- **Strengthen risk management:** Secure deployment defaults, redaction of recognized sensitive values, audit records, and automated reporting can reduce avoidable configuration, communication, and information-handling risks. Teams should still apply their organisation's security, access-control, and retention policies.

- **Increase service visibility:** Coverage reports can highlight where monitoring is in place and where there may be gaps. This provides a basis for prioritising improvement work, discussing operational exposure, and tracking progress over time.

- **Speed up incident response:** Dataview exports and ServiceNow integrations can place relevant monitoring information into established reporting and service-management workflows. Faster access to consistent information can reduce hand-offs and improve incident coordination.

- **Make better use of the existing platform:** Cordial extends existing Geneos installations and supports local and remote operations, integrations, and reusable components. Organisations can improve the value and efficiency of their current investment without replacing the monitoring platform.

- **Support scalable growth:** Standard tooling and reusable workflows make it easier to manage a growing or distributed monitoring estate with less dependence on individual scripts or specialist knowledge. The result is a more maintainable operating model as the estate and team evolve.

## Operational workflows

### Manage Geneos with one command


The [`geneos`](tools/geneos/README.md) command brings common management tasks into one consistent interface. You can use it across local and remote hosts, including with existing installations. Its main benefits include:

- **Consistent operations:** View, install, update, configure, start, and stop Geneos components with repeatable commands instead of maintaining separate scripts for each task.

- **Security by default:** New deployments use TLS. Unless you provide your own certificates or choose insecure mode, `geneos` establishes a private certificate authority and issues certificates for Geneos components. This gives you a consistent foundation for protected communications without building a separate certificate workflow.

- **Faster common setups:** Ready-made Gateway include files based on Geneos best practices, together with profiles, help you deploy common monitoring scenarios without building each configuration from scratch.

- **Operational audit:** Selected command invocations and instance actions are recorded in global and per-instance logs. You can review who ran a command, what action it took, and whether it succeeded. Recognized sensitive values are redacted from logged command details.

- **Monitoring Health (preview):** Previously known as Monitor-Of-The-Monitors (MOTM), this uses a Self-Announcing Netprobe to report installed Geneos components to a central Health Gateway. This gives you a consolidated view without defining every managed entity by hand. Rebuild the profile after estate changes to refresh that view.

### Review monitoring coverage with GDNA

Geneos licence data can be detailed and spread across many Gateways and Netprobes. [GDNA](gdna/README.md) turns data from Geneos Licence Daemons (`licd`) into dashboards and reports that show monitoring coverage across servers; it does not collect monitoring data from other sources. You and your team can use these reports to find potential gaps, compare plugin use with your coverage goals, and plan improvements. GDNA can also email reports on a schedule, so you can share regular updates without running them manually. Results depend on the licence data available and the coverage policies in use.

### Share Dataviews with dv2email

[`dv2email`](tools/dv2email/README.md) turns Geneos Dataview snapshots into email reports or files such as HTML and Excel workbooks. A Gateway Action or Effect can start a report when an alert occurs. This avoids repeated manual exports and copy-and-paste for incident reviews, handovers, and other operational reports.

### Connect alerts to ServiceNow

The [`servicenow2` integration](integrations/servicenow2/README.md) maps Geneos alert data to ServiceNow incident fields. Based on its configured rules and correlation values, it can create an incident or update an existing one. It removes the need for a separate wrapper script to translate Gateway event data. This reduces manual ticket entry and keeps alerts in your service-management workflow.

## Low-overhead deployment

The pre-built Linux command-line programs are statically linked for compatible 64-bit x86 Linux systems. You do not need to install a separate language runtime, interpreter, or helper utility to run them. The exceptions are `libemail.so` and `libalert.so`: they are shared libraries that use the host's standard C runtime and must be loaded by a compatible Geneos Gateway. Each program still needs the configuration, network access, and external services required for its work.

The `geneos` command also includes tools for many tasks that might otherwise need separate scripts or diagnostic utilities. For example, `geneos package install` can download and unpack releases; `geneos ps` can inspect running instances, open files, and sockets; `geneos logs` can read, filter, and follow local or remote logs; and `geneos snapshot` can fetch Gateway Dataviews. These built-in tools can reduce software setup and maintenance on your Geneos hosts.

## Integrations

| Integration | Purpose |
| --- | --- |
| [`servicenow2`](integrations/servicenow2/README.md) | Creates or updates ServiceNow incidents from Geneos alerts. |
| [`ims-gateway`](tools/ims-gateway/README.md) | Connects Geneos incident commands to ServiceNow or ServiceDesk Plus. |

## Libraries

| Library | Purpose |
| --- | --- |
| [`libalert`](libraries/libalert/README.md) | Geneos notification library with enhanced email support and Microsoft Teams notifications. |
| [`libemail`](libraries/libemail/README.md) | Geneos email library with support for authenticated SMTP, TLS, and templates. |

## Other tools

| Tool | Purpose |
| --- | --- |
| [`files2dv`](tools/files2dv/README.md) | Builds Geneos Toolkit-compatible Dataview data from files. Its background push mode is not implemented. |

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
