# ITRS Layer 1 Server Health

## Overview

The ITRS Layer 1 Server Health include file provides pre-configured, best practice monitoring of server CPU, disk, memory, hardware and network interfaces through a single Gateway Type, `Layer 1 - Server Health`. It is embedded in the `geneos` program, used by the `demo` and `health-gateway` profiles, and can be added to any Gateway.

Files included, relative to your Geneos installation directory, for the example site name `acme`:

| File | Purpose |
| ---- | ------- |
| `gateway/includes/itrs.layer1.server-health.xml` | ITRS monitoring definitions - read-only, replaced on update |
| `gateway/acme/acme.layer1.server-health.xml` | Your site overrides - empty by default, never overwritten |

The `itrs` file is extracted into a version directory and linked from the `includes` directory, so Gateways always reference the same path:

```text
gateway/
├── includes/
│   ├── itrs.layer1.server-health.xml -> vx.y.z/itrs.layer1.server-health.xml
│   └── vx.y.z/itrs.layer1.server-health.xml
└── acme/acme.layer1.server-health.xml
```

>[!NOTE]
>Do not edit the `itrs` file. Put changes in the site file, or in the main setup file of a single Gateway, as these have a higher include priority and override the ITRS defaults. If no site name is set, the directory and prefix are `local`. See [Gateway Setup Files](https://docs.itrsgroup.com/docs/geneos/7.12.0/processing/configuration/gateway_setup_files/index.html).

## Installation

New installations created with `geneos` set this up automatically. For existing installations, follow the upgrade steps below before updating the links:

```bash
geneos includes extract --update
```

Changed files in existing version directories are backed up. Existing site configuration files are never overwritten. To switch to another installed version use `geneos includes update --version vx.y.z`. See [`geneos includes`](../../../docs/geneos_includes.md).

### Upgrading from an earlier include

The former `itrs.level1.infrastructure.xml` and `itrs.level1.server-health.xml` includes and their matching site configurations have been renamed. Existing Gateway setups, site overrides and user profiles are not updated automatically.

1. Back up your Gateway setups and site configuration. Run `geneos includes install` without `--update` to install the new files while keeping existing links.
2. Copy or merge overrides from your old site configuration into the new `layer1.server-health.xml` site file, preserving any settings already there. Update references to the old `Level 1 - Infrastructure` or `Level 1 - Server Health` types, groups, environments and XPath targets in those overrides.
3. Update Gateway include paths and any type, group and environment references in Gateway setups, customised profiles and SAN configurations to the new Server Health names.
4. Once no Gateway setup references the old include link, run `geneos includes update`. The update refuses to remove an old link that is still in use.

## Usage

>[!NOTE]
>If you have used a profile to install Geneos then the include is most likely already configured and you can go to step 2.

1. Add both files to your Gateway, either with `geneos set gateway NAME -i ...` (see [gateway](../../../internal/component/gateway/gateway.md)) or by pasting this XML under `Includes` in the Gateway Setup Editor (GSE):

   ```xml
   <includeGroup name="ITRS Layer 1 - Server Health">
       <include>
           <priority>1000</priority>
           <required>true</required>
           <location>../../includes/itrs.layer1.server-health.xml</location>
       </include>
       <include>
           <priority>500</priority>
           <required>true</required>
           <location>../../acme/acme.layer1.server-health.xml</location>
       </include>
   </includeGroup>
   ```

2. Attach the Type to your Managed Entities, either by placing them in a Managed Entity Group named `Layer 1 - Server Health`, or by adding the Type directly:

   | Type | Environment |
   | ---- | ----------- |
   | `Layer 1 - Server Health` | `Layer 1 - Server Health Overrides` |

   For a Self-Announcing Netprobe (SAN), use `-t 'Layer 1 - Server Health'`.

To change the defaults, define the variables to override in the `Layer 1 - Server Health Overrides` environment (in the `Layer 1 - Server Health` environment group) in your `config` file. Samplers and rules can be overridden by redefining them with the same name and group. Redefined items replace the ITRS version completely, so copy the full item before changing it. See [User Variables and Environments](https://docs.itrsgroup.com/docs/geneos/6.0.0/processing/gateway-administration/gateway_user_variables_and_environments/).

## Contents

### Samplers

| Sampler | Plugin | Interval | Notes |
| ------- | ------ | -------- | ----- |
| CPU | [cpu](https://docs.itrsgroup.com/docs/geneos/7.4.0/collection/cpu/index.html) | `FAST_SAMPLE` | |
| Disk | [disk](https://docs.itrsgroup.com/docs/geneos/7.4.0/collection/disk/index.html) | `SLOW_SAMPLE` | Auto-detects local partitions; excludes pseudo, virtual and removable file systems; NFS not checked |
| Hardware | [hardware](https://docs.itrsgroup.com/docs/geneos/7.4.0/collection/hardware/) | `NORMAL_SAMPLE` | |
| Network | [network](https://docs.itrsgroup.com/docs/geneos/7.4.0/collection/network/index.html) | `FAST_SAMPLE` | Excludes loopback and unconfigured NICs; rates in Mbits |

### Variables

| Variable | Default | Purpose |
| -------- | ------- | ------- |
| `FAST_SAMPLE` | `5` | CPU and Network sample interval (seconds) |
| `NORMAL_SAMPLE` | `20` | Hardware sample interval (seconds) |
| `SLOW_SAMPLE` | `300` | Disk sample interval (seconds) |
| `EXPECTED_PARTITIONS` | empty | Partitions expected in the Disk view |

### Rules

| Rule | Warning | Critical |
| ---- | ------- | -------- |
| CPU Utilisation | > 80% | > 90% |
| Disk Percentage Used | > 85% and free < 10000, or > 80% and free < 20000 | > 90% and free < 10000 |
| Memory Used Percent | > 80% | > 90% |
| Swap Used | > 90% | - |
| Zombie Process (Unix) | > 10 | - |
| Network Status (Unix) | not `OK` | `DOWN` |
| TCP Retransmission Rate (Windows) | > 20/s | - |
| Paging | - | - |

Rules with no threshold use a placeholder value (`999999`) that you can override with a realistic value. Managed Entities in the group get the attributes `CATEGORY=Infrastructure` and `COMPONENT=System`. Samplers appear under the `System` sampler group.
