# Level 3 Geneos Configuration Example

`config.level3.geneos.xml` contains an example managed-entity hierarchy for the types defined by `itrs.level3.geneos.xml`. It is a configuration template, not a live monitoring definition.
For the site name `acme`, the installed file is `gateway/acme/acme.level3.geneos.xml`; if the site name is unset, it is `gateway/local/local.level3.geneos.xml`.

## Example Hierarchy

- In the Gateway Setup Editor (GSE), the `Level 3 - Geneos` managed-entity group sets the `APPLICATION`, `CATEGORY`, and `COMPONENT` attributes and adds the `Level 3 - Geneos Instances` type with the `Level 3 - Geneos Overrides` environment.
- The included managed entity named `example` is disabled and points to the `localhost` probe as a placeholder. It also shows the five per-entity variables required when component monitoring types are assigned: `instanceName`, `instanceType`, `instanceExecPath`, `logFilePath`, and `consoleLogFilePath`.
- The nested `With Diagnostic Views` group adds `Level 3 - Geneos Diagnostics`, also with the shared overrides environment.
- After the managed entities, the file defines empty copies of `Level 3 - Geneos Overrides` and `Level 3 - Geneos Load Overrides` under their corresponding environment groups. Add site-specific values to these copies in the GSE.

## Usage

Use this example together with `itrs.level3.geneos.xml`, which defines the referenced types, samplers, and sampling Environments. In the GSE, adapt the hierarchy for the target Gateway: create or update the managed-entity groups, set their attributes, add the appropriate entities, and connect them to their actual Netprobes. For component monitoring, set all five required instance variables on each managed entity, either manually or through a profile; the functional include no longer supplies Type-level fallbacks. Set global `geneosExec*` variables under the Gateway's `operatingEnvironment`. The XML is provided as a template/reference; these instructions do not require hand-editing XML.

## Include Priority and Upgrades

In the GSE, give `config.level3.geneos.xml` a numerically lower priority (therefore higher precedence) than the read-only `itrs.level3.geneos.xml`. For example, if the functional include has priority 300, assign the config include a unique priority below 300. When settings conflict during Gateway include merging, the higher-priority config values take precedence.

Keep `itrs.level3.geneos.xml` read-only and upgrade it from the published version when updating the monitoring functionality. Put Gateway-specific changes in the config include so they remain separate from the upgradeable definitions.

The parent group automatically applies instance monitoring to entities placed under it. Place an entity in `With Diagnostic Views` only when open-file and TCP-port diagnostics are wanted. For component monitoring, also assign `Level 2 - Basic Application` from `itrs.level2.basic-application.xml` and the relevant Level 3 component type to that entity; the component sampler includes extend the Level 2 `Logs` and `Processes` samplers. Add Gateway-load types separately where required.

Do not enable the placeholder entity unchanged. Replace or remove it as part of adapting the example; keep the template separate from production configuration when no example hierarchy is needed.
