# Layer 3 Geneos Configuration Example

`config.layer3.geneos.xml` contains an example Managed Entity hierarchy for the Types defined by `itrs.layer3.geneos.xml`. It is a configuration template, not a live monitoring definition.

For the site name `acme`, the installed file is `gateway/acme/acme-config.layer3.geneos.xml`; if the site name is unset, it is `gateway/local/local-config.layer3.geneos.xml`. The file is copied only when absent and is never overwritten by later installations.

## Example Hierarchy

- In the Gateway Setup Editor (GSE), the `Layer 3 - Geneos` Managed Entity group sets the attributes `APPLICATION=Geneos`, `CATEGORY=Information` and `COMPONENT=Geneos`, and adds the `Layer 3 - Geneos Instances` Type with the `Layer 3 - Geneos Local` Environment.

- The included Managed Entity named `example` is disabled and points to the `localhost` probe as a placeholder. It shows the five per-entity variables used by the `Layer 3 - Geneos Component` Type: `instanceName`, `instanceType`, `instanceExecPath`, `logFilePath` and `consoleLogFilePath`.

- The nested `With Diagnostic Views` group adds `Layer 3 - Geneos Instances Diagnostics`, also with the `Layer 3 - Geneos Local` Environment. Place a Managed Entity in this group to add the open files and open TCP ports views.

- After the Managed Entities, the file defines empty `Layer 3 - Geneos Local` and `Layer 3 - Geneos Load Local` Environments under the `Layer 3 - Geneos` and `Layer 3 - Geneos Gateway Load` environment groups. Add site-specific values to these Environments in the GSE. There is no matching Environment for the `Layer 3 - Geneos Gateway Info` Type.

## Usage

Use this example together with `itrs.layer3.geneos.xml`, which defines the referenced Types, samplers and default Environments. In the GSE, adapt the hierarchy for the target Gateway: create or update the Managed Entity groups, set their attributes, add the appropriate entities, and connect them to their actual Netprobes.

For component monitoring, assign `Layer 2 - Application Baseline` from `itrs.layer2.programs-and-logs.xml` and `Layer 3 - Geneos Component` to the Managed Entity, and set all five instance variables on it, either manually or through a profile; the functional include supplies no Type-level defaults. For a Netprobe, also assign `Layer 3 - Geneos Netprobe Additions` and set `caLogFilePath`. The component sampler includes extend the Layer 2 `Logs` and `Processes` samplers.

Add the Gateway load and Gateway Info Types separately where required. The `Layer 3 - Geneos Load Local` Environment is not attached to anything in the example; reference it from the Gateway load Type assignment if you use it.

Set the global `geneosExec*` variables under the Gateway's `operatingEnvironment`. The XML is provided as a template and reference; these instructions do not require hand-editing XML.

## Include Priority and Upgrades

In the GSE, give `config.layer3.geneos.xml` a numerically lower priority (therefore higher precedence) than the read-only `itrs.layer3.geneos.xml`. The `health-gateway` profile, for example, uses priority `1020` for the functional include and `520` for the site file. When settings conflict during Gateway include merging, the higher-precedence site values take effect.

Keep `itrs.layer3.geneos.xml` read-only and upgrade it from the published version when updating the monitoring functionality. Put Gateway-specific changes in the site file so they remain separate from the upgradeable definitions.

Do not enable the placeholder entity unchanged. Replace or remove it as part of adapting the example; keep the template separate from production configuration when no example hierarchy is needed.

Earlier versions of this file used `Layer 3 - Geneos Overrides` and `Layer 3 - Geneos Load Overrides` Environments and the `Layer 3 - Geneos Diagnostics` Type. If an existing site file or Gateway setup still references these names, update the references to `Layer 3 - Geneos Local`, `Layer 3 - Geneos Load Local` and `Layer 3 - Geneos Instances Diagnostics`.
