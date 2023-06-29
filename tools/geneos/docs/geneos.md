# `geneos`

Take control of your Geneos environments

## Subsystems

* [`geneos aes`](geneos_aes.md)	 - AES256 Key File Operations
* [`geneos config`](geneos_config.md)	 - Configure Command Behaviour
* [`geneos host`](geneos_host.md)	 - Remote Host Operations
* [`geneos init`](geneos_init.md)	 - Initialise The Installation
* [`geneos package`](geneos_package.md)	 - Package Operations
* [`geneos tls`](geneos_tls.md)	 - TLS Certificate Operations

## Control Instances

* [`geneos reload`](geneos_reload.md)	 - Reload Instance Configurations
* [`geneos restart`](geneos_restart.md)	 - Restart Instances
* [`geneos start`](geneos_start.md)	 - Start Instances
* [`geneos stop`](geneos_stop.md)	 - Stop Instances

## Inspect Instances

* [`geneos command`](geneos_command.md)	 - Show Instance Start-up Details
* [`geneos home`](geneos_home.md)	 - Display Instance and Component Home Directories
* [`geneos list`](geneos_list.md)	 - List Instances
* [`geneos logs`](geneos_logs.md)	 - View Instance Logs
* [`geneos ps`](geneos_ps.md)	 - List Running Instance Details
* [`geneos show`](geneos_show.md)	 - Show Instance Configuration

## Manage Instances

* [`geneos clean`](geneos_clean.md)	 - Clean-up Instance Directories
* [`geneos copy`](geneos_copy.md)	 - Copy instances
* [`geneos disable`](geneos_disable.md)	 - Disable instances
* [`geneos enable`](geneos_enable.md)	 - Enable instance
* [`geneos move`](geneos_move.md)	 - Move instances
* [`geneos protect`](geneos_protect.md)	 - Mark instances as protected

## Configure Instances

* [`geneos add`](geneos_add.md)	 - Add a new instance
* [`geneos delete`](geneos_delete.md)	 - Delete Instances
* [`geneos deploy`](geneos_deploy.md)	 - Deploy a new Geneos instance
* [`geneos import`](geneos_import.md)	 - Import Files To Instances Or Components
* [`geneos migrate`](geneos_migrate.md)	 - Migrate Instance Configurations
* [`geneos rebuild`](geneos_rebuild.md)	 - Rebuild Instance Configurations From Templates
* [`geneos revert`](geneos_revert.md)	 - Revert Migrated Instance Configuration
* [`geneos set`](geneos_set.md)	 - Set Instance Parameters
* [`geneos unset`](geneos_unset.md)	 - Unset Instance Parameters

## Manage Credentials

* [`geneos login`](geneos_login.md)	 - Enter Credentials
* [`geneos logout`](geneos_logout.md)	 - Remove Credentials

## Miscellaneous

* [`geneos snapshot`](geneos_snapshot.md)	 - Capture a snapshot of each matching dataview
* [`geneos version`](geneos_version.md)	 - Show program version

## Recognised Component Types

* [`geneos ca3`](geneos_ca3.md)	 - Collection Agent 3
* [`geneos fa2`](geneos_fa2.md)	 - Fix Analyser 2
* [`geneos fileagent`](geneos_fileagent.md)	 - File Agent
* [`geneos floating`](geneos_floating.md)	 - Floating Netprobes
* [`geneos gateway`](geneos_gateway.md)	 - Gateways
* [`geneos licd`](geneos_licd.md)	 - Licence Daemon
* [`geneos netprobe`](geneos_netprobe.md)	 - Netprobes
* [`geneos san`](geneos_san.md)	 - Self-Announcing Netprobes
* [`geneos webserver`](geneos_webserver.md)	 - Web Dashboard Servers

The `geneos` program will help you manage your Geneos environment.

With `geneos` you can initialise a new installation, install and update software releases, add and remove instances, control processes and build template based configuration files for SANs and more.

The subsystems group the management of related functions together.

### Options

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, geneos/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## Examples

```bash
geneos init demo -u jondoe@example.com -l
geneos ps
geneos restart

```

## SEE ALSO

