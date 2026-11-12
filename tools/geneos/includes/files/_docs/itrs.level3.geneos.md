# ITRS Level 3 Geneos Documentation

`itrs.level3.geneos.xml` defines reusable Geneos instance, component, and Gateway-load types and samplers, component process/log includes, commands, alert rules, and a data export set. It contains no managed-entity hierarchy; the matching `config.level3.geneos.xml` provides an example configuration.

Treat this functional include as read-only so it can be upgraded from later releases without losing local changes. Keep managed-entity setup and site-specific overrides in the matching `config.level3.geneos.xml` or your Gateway's local configuration.

## Dependencies and Installation

The instance, diagnostics, and Gateway-load types are defined here and do not depend on the Level 0 Geneos types or the Level 2 application type. The `Level 3 - Geneos Components` types do require the Level 2 `Level 2 - Basic Application` type because their sampler includes extend its `Logs` and `Processes` samplers.

For component monitoring, include `itrs.level2.basic-application.xml` and assign both `Level 2 - Basic Application` and the relevant `Level 3 - Geneos Components` type to the managed entity. `Level 3 - Geneos Component Logs` adds the component log files to `Logs`; `Level 3 - Geneos Component Process` adds the component process descriptor to `Processes`.

Use `config.level3.geneos.xml` as a template for a managed-entity hierarchy. Its `Level 3 - Geneos` group adds `Level 3 - Geneos Instances`; entities in the nested `With Diagnostic Views` group also receive `Level 3 - Geneos Diagnostics`. The example managed entity is disabled. Adapt the hierarchy and connect real entities to the appropriate Netprobe in your Gateway configuration. For component monitoring, add the Level 2 base type and relevant Level 3 component type to the entity; assign Gateway-load types separately where needed.

## Type Entrypoints

| Type | Use | Samplers / behavior |
| --- | --- | --- |
| Level&nbsp;3&nbsp;-&nbsp;Geneos&nbsp;Instances | Monitor the Geneos installation on a host. | Runs queries with the local `geneos` command for AES files, certificates, installed releases, configured instances, running processes, version summary, and toolkit include files. |
| Level&nbsp;3&nbsp;-&nbsp;Geneos&nbsp;Diagnostics | Add process-level diagnostics. | Reports open files and open TCP ports. Use where the additional detail is useful; the port rules warn on unexpected connection states and growing queues. |
| Level&nbsp;3&nbsp;-&nbsp;Geneos&nbsp;Active&nbsp;Console | Monitor an Active Console instance. | Adds component-specific logs and process monitoring through sampler includes used with the Level 2 `Logs` and `Processes` samplers. |
| Level&nbsp;3&nbsp;-&nbsp;Geneos&nbsp;Floating&nbsp;Netprobe | Monitor a floating Netprobe instance. | Adds component-specific logs and process monitoring through sampler includes used with the Level 2 `Logs` and `Processes` samplers. |
| Level&nbsp;3&nbsp;-&nbsp;Geneos&nbsp;Gateway | Monitor a Gateway process and logs. | Adds component-specific logs and process monitoring through sampler includes used with the Level 2 `Logs` and `Processes` samplers. |
| Level&nbsp;3&nbsp;-&nbsp;Geneos&nbsp;Licence&nbsp;Daemon | Monitor a Licence Daemon instance. | Adds component-specific logs and process monitoring through sampler includes used with the Level 2 `Logs` and `Processes` samplers. |
| Level&nbsp;3&nbsp;-&nbsp;Geneos&nbsp;Netprobe | Monitor a Netprobe instance. | Adds component-specific logs and process monitoring through sampler includes used with the Level 2 `Logs` and `Processes` samplers. |
| Level&nbsp;3&nbsp;-&nbsp;Geneos&nbsp;Netprobe-Router | Monitor a Netprobe-Router instance. | Adds component-specific logs and process monitoring through sampler includes used with the Level 2 `Logs` and `Processes` samplers. |
| Level&nbsp;3&nbsp;-&nbsp;Geneos&nbsp;Self&nbsp;Announcing&nbsp;Netprobe | Monitor a self-announcing Netprobe instance. | Adds component-specific logs and process monitoring through sampler includes used with the Level 2 `Logs` and `Processes` samplers. |
| Level&nbsp;3&nbsp;-&nbsp;Geneos&nbsp;SSO-Agent | Monitor an SSO-Agent instance. | Adds component-specific logs and process monitoring through sampler includes used with the Level 2 `Logs` and `Processes` samplers. |
| Level&nbsp;3&nbsp;-&nbsp;Geneos&nbsp;Webserver | Monitor a Webserver instance. | Adds component-specific logs and process monitoring through sampler includes used with the Level 2 `Logs` and `Processes` samplers. |
| Level&nbsp;3&nbsp;-&nbsp;Geneos&nbsp;Gateway&nbsp;Load | Inspect Gateway load from the Gateway statistics file. | Components, connections, rules grouped by several dimensions, database logging, Gateway SQL, rule threads, and data-quality history. `Rules by Sampler and Rule` and `Rules by Entity-Rule` are disabled in the type. |
| Level&nbsp;3&nbsp;-&nbsp;Geneos&nbsp;Gateway&nbsp;Load&nbsp;XPaths | Inspect Gateway XPath load. | Absolute and relative XPath statistics. |

The instance and diagnostic types set `SAMPLER_GROUP` to `Geneos Instances`; the Gateway-load types set it to `Gateway Load`. Component types attach the same two sampler includes, whose log and process selection is driven by variables on each managed entity.

## Variables

### Global Geneos Command Variables

The `geneos` command settings are global variables under `operatingEnvironment`, not Type or Environment defaults. Change them there to affect toolkit queries in both the Instances and Diagnostics samplers.

| Variable | Default | Effect |
| --- | --- | --- |
| `geneosExecPath` | `/usr/local/bin/geneos` | Executable used by the toolkit samplers. |
| `geneosExecConfigPath` | A single space | Optional `geneos` command configuration argument. Preserve the space when no argument is needed; an empty value can produce malformed command lines. |
| `geneosExecHosts` | `-H localhost` | Host argument passed to the `geneos` command queries. |

### Sampling Defaults and Tuning

`Level 3 - Geneos Defaults` is used by the instance, diagnostics, and component types. It provides sample intervals only:

| Variable | Default | Effect |
| --- | --- | --- |
| `SLOW_SAMPLE` | `300` seconds | Interval for slower inventory and summary samplers. |
| `NORMAL_SAMPLE` | `20` seconds | Interval for running-process and diagnostic samplers. |
| `FAST_SAMPLE` | `5` seconds | Defined for reuse; not used by the samplers in this include. |

Certificates are an exception to the interval defaults: they run once daily at 01:00.

`Level 3 - Geneos Load Defaults` is used by the Gateway-load types:

| Variable | Default | Effect |
| --- | --- | --- |
| `SLOW_SAMPLE`, `NORMAL_SAMPLE`, `FAST_SAMPLE` | `300`, `20`, `5` seconds | Sampling intervals; the Gateway-load samplers here use `NORMAL_SAMPLE`. |
| `gwLoadHistoryColumns` | `cpu1Min`, `cpu5Min` | Extra calculated history columns added to applicable views. |
| `gwLoadDatabaseColumns` | `cpu1Min`, `cpu5Min`, `updates1Min`, `updates5Min` | Extra columns for database-logging views. |
| `gwLoadEnvHistory Columns` | `lastHour`, `lastDay` | Extra calculated columns for the Data Quality view. Keep the variable name's embedded space exactly as written. |
| `statsFilePath` | `stats.xml` | Statistics file path used by Gateway-load samplers; override it on a Gateway entity if the file is elsewhere. |

### Per-Entity Variables

Component Types have no defaults for the following variables. Set them directly on every managed entity that uses a component Type, either manually in Gateway Setup Editor or through a profile. Do not rely on an Environment or Type-level fallback; variable scope can prevent those values from reaching the component samplers. The disabled example entity in `config.level3.geneos.xml` shows one set of variables. Gateway-load types continue to default `statsFilePath` to `stats.xml`; override it on a Gateway entity when its statistics file is elsewhere.

| Variable | Example value | Used for |
| --- | --- | --- |
| `instanceName` | `example` | Process descriptor alias and process matching; set to the actual instance name. |
| `instanceType` | `gateway` | Process descriptor alias and start/stop command arguments; set to the component key such as `gateway` or `netprobe`. |
| `instanceExecPath` | `/usr/local/bin/geneos` | Matching the running process to the instance. |
| `logFilePath` | `gateway.log` | Component log-file monitoring; set to the actual log path. |
| `consoleLogFilePath` | `gateway.txt` | Console-log monitoring; set to the actual console-log path. |
| `statsFilePath` | `stats.xml` | Gateway-load statistics file path. |

The values shown are examples, not defaults supplied by the functional include. Set them to match each monitored instance. The `statsFilePath` default is the relative filename `stats.xml`; override it on the Gateway entity if that file is not accessible from the monitoring Netprobe.

## Override Environments

The include declares sampling defaults and empty override Environment placeholders. Use Environments for shared sample intervals and other scoped settings; the `geneosExec*` variables are global in `operatingEnvironment`, while component process and log variables must be set on each managed entity. An Environment does not add samplers or change the type's rule logic. The trees below are conceptual Setup Editor views; their top-level order follows Gateway XML, with managed entities before environment definitions.

### `Level 3 - Geneos Overrides`

The example hierarchy in `config.level3.geneos.xml` attaches this environment to `Level 3 - Geneos Instances` and, in the nested subgroup, `Level 3 - Geneos Diagnostics`. The config file contains an empty copy under the same environment group; add shared override values to that Environment in Gateway Setup Editor. To use different settings for one type or entity, attach a custom Environment to that Type assignment instead.

For example, to change the sample intervals, use Gateway Setup Editor to add or edit an Environment under the `Level 3 - Geneos` environment group, then select it on the managed entity's `Level 3 - Geneos Instances` Type assignment. Set global command variables under the Gateway's `operatingEnvironment` instead:

```text
Gateway Setup Editor
├─ Managed entities
│  └─ Managed entity
│     └─ Type: Level 3 - Geneos Instances
│        └─ Environment assignment: Level 3 - Geneos Overrides
└─ Environments
   └─ Environment group: Level 3 - Geneos
      └─ Environment: Level 3 - Geneos Overrides
         ├─ NORMAL_SAMPLE = 30
         └─ SLOW_SAMPLE = 600
```

Define only the sampling variables to change. The same Environment can be attached to `Level 3 - Geneos Diagnostics`.

### `Level 3 - Geneos Load Overrides`

This placeholder is declared but not referenced by either Gateway-load type. The matching config file contains an empty copy under the same environment group. Attach it explicitly to a Gateway-load Type assignment and add variables such as `gwLoadHistoryColumns`, `gwLoadDatabaseColumns`, or sample intervals in Gateway Setup Editor. Override `statsFilePath` on the Gateway entity if the monitoring Netprobe cannot access the file as `stats.xml`.

For example, in Gateway Setup Editor, add an Environment under the Gateway-load environment group, set the desired columns, then select it on the Gateway entity's `Level 3 - Geneos Gateway Load` Type assignment:

```text
Gateway Setup Editor
├─ Managed entities
│  └─ Gateway managed entity
│     └─ Type: Level 3 - Geneos Gateway Load
│        └─ Environment assignment: Level 3 - Geneos Load Overrides
└─ Environments
   └─ Environment group: Level 3 - Geneos Gateway Load
      └─ Environment: Level 3 - Geneos Load Overrides
         └─ gwLoadHistoryColumns = cpu1Min, cpu5Min, cpuHour
```

Use the corresponding `Level 3 - Geneos Gateway Load XPaths` type when XPath statistics are needed; it uses the same load defaults and can use the same override environment.

## Operational Behavior

- Instance samplers call the local `geneos` command by default. `geneosExecPath`, `geneosExecConfigPath`, and `geneosExecHosts` control the assembled command invocation. Inventory samplers normally run every five minutes; process status normally runs every 20 seconds.
- Certificate checks run daily and rules warn on invalid certificates, certificates expiring within 30 days, and critically on those expiring within 7 days.
- Rules also flag old installed releases, disabled instances, non-TLS instances, version mismatches, and increasing open-file/socket counts. These severities are rule behavior, not environment-variable settings.
- Component Logs uses FKM for the component log and optional console log. The Processes sampler matches the configured process descriptor; a missing process is critical and multiple matches are warning-level.
- Gateway-load history rules calculate rolling CPU and database update columns from successive samples. Gateway-load views require Gateway statistics collection to be enabled and `statsFilePath` to point to the resulting file.
- The include provides `Restart NetProbe`, `View Configuration for Dataview`, and `View Gateway Logfile` commands, plus a `Level 3 - Geneos` exported data set.

## Review Notes

- The exported-data path for `Level 3 - Geneos Instance` is singular, but the defined type is `Level 3 - Geneos Instances`. Check whether that path should use the plural type name so the instance sampler data is exported as intended.
