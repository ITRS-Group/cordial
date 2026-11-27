# Layer 0 Authentication Configuration

## Overview

`config.layer0.authentication.xml` is a Gateway setup template containing example authentication settings: two roles and two user groups with example users. It is a starting point for local configuration, not a functional include. Every user in the file is disabled.

>[!WARNING]
>The users in this file are examples. Review every identity and replace or remove them before enabling authentication. Never enable or use the example passwords; set new passwords, or use system or external authentication, for any account you keep.

For the example site name `acme`, the file is installed as:

| File | Purpose |
| ---- | ------- |
| `gateway/acme/acme-config.layer0.authentication.xml` | Site authentication settings - copied once, never overwritten |

If no site name is set, the directory and prefix are `local`. Later installations do not overwrite the file, so your changes are kept. No `geneos` profile adds this file to a Gateway.

## Usage

1. Edit the site file, or copy its settings into the main setup file. Replace the example users with your own, or remove them.

2. Add the file to your Gateway, either with `geneos set gateway NAME -i PRIORITY:PATH` or by pasting this XML under `Includes` in the Gateway Setup Editor (GSE):

   ```xml
   <includeGroup name="ITRS Layer 0 - Authentication">
       <include>
           <priority>4000</priority>
           <required>true</required>
           <location>../../acme/acme-config.layer0.authentication.xml</location>
       </include>
   </includeGroup>
   ```

   Include priorities must be unique and must fit the Gateway's include hierarchy.

3. Enable authentication on the Gateway and test that administrators can still log in before relying on it.

Restrict access to the file, as it can contain user and password settings. If the Gateway uses an AES key file for encrypted passwords, use the same key file on all Gateways that share the file.

## Contents

### Roles

| Role | Role property | Permissions |
| ---- | ------------- | ----------- |
| `Administrators` | `Administrators` | No explicit permissions; the `Administrators` role property gives full administrative access |
| `ReadOnly` | `ReadOnly` | No access to any command on any target (`/*`, all command names); no access to the main setup file or any include file, with no locking |

### User Groups

| User group | Role property | Users |
| ---------- | ------------- | ----- |
| `Administrators` | `Administrators` | Two example named domain users and one generic user |
| `ReadOnly` | `ReadOnly` | One example Gateway sharing account |

All users are disabled (`disabled="true"`).

- The example named users in the `Administrators` group have full names, are allowed to log in and are allowed system authentication. They have no password in the file.

- The generic user in the `Administrators` group is marked as a generic user and is allowed to log in with system authentication. A generic user is used for system-authenticated users that have no individual user definition, so if enabled it gives such users the group's administrative role. Remove it unless that is what you intend.

- The `GWSharing` user in the `ReadOnly` group is intended as a Gateway sharing account. It has a local password and the `ReadOnly` role property. Set a new password before enabling it.

## Upgrades

The template is installed only when the site file does not already exist. Compare your site file with new releases of the template and merge any changes you need.

## Review Notes

- The template contains a comment that states the example password for `GWSharing` in clear text. Remove the comment from your site file and set a new password before enabling the account.

- The template has no authentication policies; only roles and user groups are defined.
