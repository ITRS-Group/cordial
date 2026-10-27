# ITRS Level 1 Infrastructure

## Overview

The ITRS Level 1 Infrastructure include file provides pre-configured monitoring setups for Geneos Gateways, enabling efficient tracking and management of Level 1 infrastructure components. These files are maintained by ITRS and are intended to be used as-is, with local customizations applied through corresponding configuration files.

## Files Included

>[!NOTE]
>All include files with an `itrs` prefix are created and maintained by ITRS and should be treated as read-only and not edited directly. This way we can release updated versions in future releases and these can be safely applied without overwriting local customizations. To make changes to suit your local requirements, follow the directions below.
>
>The corresponding `config` file below is a good place to put your local customizations and overrides, ensuring that your changes are applied without modifying the original ITRS-maintained include files. If you need to add overrides for just one Gateway then these should go in the main Gateway configuration file for that Gateway.

Files included in this group, relative to your Geneos installation directory:

- `gateway/includes/itrs.level1.infrastructure.xml`
- `gateway/includes/config/config.level1.infrastructure.xml`

## Installation

These files are embedded in the `geneos` program, shipped as part of the `cordial` tools. Unless you are setting up a new installation, in which case this next step is automatic, you will need to extract the files and also update the version-specific links to point to the correct locations for your installation. To do this run the following:

```bash
geneos includes extract --update
```

This commands will not overwrite existing real files stored in versioned directories, unless also give the `--force` option. It will update symbolic links to point to the latest extracted versions however..

## Usage

>[!NOTE]
>If you have used a "profile" to install Geneos then this include file is most likely already included and you can skip the step below.

Use the file in your Gateway configuration like this (copy the text below, hover over `Includes` in the GSE and select `Paste XML`):

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
        <location>../../includes/config/config.level1.infrastructure.xml</location>
    </include>
</includeGroup>
```

To use the features included load it into your Gateway set-up, as above, and then add Probes and Managed Entities with the following Types and Environments linked:

| Type                     | Environment                        |
| ------------------------ | ---------------------------------- |
| Level 1 - Infrastructure | Level 1 - Infrastructure Overrides |

The `Level 1 - Infrastructure Overrides` environment is empty by default and to add you own settings you should define your own environment in the right Environment Group, as below, so that the Gateway will correctly apply your configuration over the underlying settings in the lower-priority include file.