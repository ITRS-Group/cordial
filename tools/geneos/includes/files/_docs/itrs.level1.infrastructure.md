# ITRS Level 1 Infrastructure

## Overview

The ITRS Level 1 Infrastructure include file provides pre-configured, best practice monitoring of basic server infrastructure - CPU, disk, memory / hardware and network - through a single Gateway Type, `Level 1 - Infrastructure`. It is embedded in the `geneos` program, used by the `demo` and `motm-gateway` profiles, and can be added to any Gateway.

Files included, relative to your Geneos installation directory, for the example site name `acme`:

| File | Purpose |
| ---- | ------- |
| `gateway/includes/itrs.level1.infrastructure.xml` | ITRS monitoring definitions - read-only, replaced on update |
| `gateway/acme/acme.level1.infrastructure.xml` | Your site overrides - empty by default, never overwritten |

The `itrs` file is extracted into a version directory and linked from the `includes` directory, so Gateways always reference the same path:

```text
gateway/
├── includes/
│   ├── itrs.level1.infrastructure.xml -> vx.y.z/itrs.level1.infrastructure.xml
│   └── vx.y.z/itrs.level1.infrastructure.xml
└── acme/acme.level1.infrastructure.xml
```

>[!NOTE]
>Do not edit the `itrs` file. Put changes in the site file, or in the main setup file of a single Gateway, as these have a higher include priority and override the ITRS defaults. If no site name is set, the directory and prefix are `local`. See [Gateway Setup Files](https://docs.itrsgroup.com/docs/geneos/7.12.0/processing/configuration/gateway_setup_files/index.html).

## Installation

New installations created with `geneos` set this up automatically. After upgrading `geneos`, extract the new version and update the links:

```bash
geneos includes extract --update
```

Existing version directories are only overwritten with `--force` (changed files are backed up first). `config` files are never overwritten. To switch to another extracted version use `geneos includes update --version vx.y.z`. See [`geneos includes`](../../../docs/geneos_includes.md).

## Usage

>[!NOTE]
>If you have used a profile to install Geneos then the include is most likely already configured and you can go to step 2.

1. Add both files to your Gateway, either with `geneos set gateway NAME -i ...` (see [gateway](../../../internal/component/gateway/gateway.md)) or by pasting this XML under `Includes` in the Gateway Setup Editor (GSE):

   ```xml
   <includeGroup name="ITRS Level 1 - Infrastructure">
       <include>
           <priority>1000</priority>
           <required>true</required>
           <location>../../includes/itrs.level1.infrastructure.xml</location>
       </include>
       <include>
           <priority>500</priority>
           <required>true</required>
           <location>../../acme/acme.level1.infrastructure.xml</location>
       </include>
   </includeGroup>
   ```

2. Attach the Type to your Managed Entities, either by placing them in a Managed Entity Group named `Level 1 - Infrastructure`, or by adding the Type directly:

   | Type | Environment |
   | ---- | ----------- |
   | `Level 1 - Infrastructure` | `Level 1 - Infrastructure Overrides` |

   For a Self-Announcing Netprobe (SAN), use `-t 'Level 1 - Infrastructure'`.

To change the defaults, define the variables to override in the `Level 1 - Infrastructure Overrides` environment (in the `Level 1 - Infrastructure` environment group) in your `config` file. Samplers and rules can be overridden by redefining them with the same name and group. Redefined items replace the ITRS version completely, so copy the full item before changing it. See [User Variables and Environments](https://docs.itrsgroup.com/docs/geneos/6.0.0/processing/gateway-administration/gateway_user_variables_and_environments/).

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
