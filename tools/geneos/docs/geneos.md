# `geneos`

Take control of your Geneos environments

## Subsystems

* [`geneos aes`](geneos_aes.md)	 - Manage Geneos compatible key files and encode/decode passwords
* [`geneos config`](geneos_config.md)	 - Configure the command environment
* [`geneos host`](geneos_host.md)	 - Manage remote host settings
* [`geneos init`](geneos_init.md)	 - Initialise a Geneos installation
* [`geneos package`](geneos_package.md)	 - Package commands subsystem
* [`geneos tls`](geneos_tls.md)	 - Manage certificates for secure connections

## Control Geneos Instances

* [`geneos reload`](geneos_reload.md)	 - Reload configurations
* [`geneos restart`](geneos_restart.md)	 - Restart instances
* [`geneos start`](geneos_start.md)	 - Start instances
* [`geneos stop`](geneos_stop.md)	 - Stop instances

## Inspect Geneos instances

* [`geneos command`](geneos_command.md)	 - Show command line and environment for instances
* [`geneos home`](geneos_home.md)	 - Output a directory path for given options
* [`geneos list`](geneos_list.md)	 - List instances
* [`geneos logs`](geneos_logs.md)	 - View, search or follow logs
* [`geneos ps`](geneos_ps.md)	 - Show running instances
* [`geneos show`](geneos_show.md)	 - Show instance configuration

## Manage Geneos Instances

* [`geneos clean`](geneos_clean.md)	 - Clean-up instance directories
* [`geneos copy`](geneos_copy.md)	 - Copy instances
* [`geneos disable`](geneos_disable.md)	 - Disable instances
* [`geneos enable`](geneos_enable.md)	 - Enable instance
* [`geneos move`](geneos_move.md)	 - Move instances
* [`geneos protect`](geneos_protect.md)	 - Mark instances as protected

## Configure Geneos Instances

* [`geneos add`](geneos_add.md)	 - Add a new instance
* [`geneos delete`](geneos_delete.md)	 - Delete instances
* [`geneos import`](geneos_import.md)	 - Import files to an instance or a common directory
* [`geneos migrate`](geneos_migrate.md)	 - Migrate configurations
* [`geneos rebuild`](geneos_rebuild.md)	 - Rebuild instance configuration files
* [`geneos revert`](geneos_revert.md)	 - Revert earlier migration of configuration files
* [`geneos set`](geneos_set.md)	 - Set instance configuration parameters
* [`geneos unset`](geneos_unset.md)	 - Unset configuration parameters

## Manage Credentials

* [`geneos login`](geneos_login.md)	 - Store credentials related to Geneos
* [`geneos logout`](geneos_logout.md)	 - Logout (remove credentials)

## Recognised Component Types

* [`geneos ca3`](geneos_ca3.md)	 - Help for ca3
* [`geneos fa2`](geneos_fa2.md)	 - Help for Fix Analyser 2
* [`geneos fileagent`](geneos_fileagent.md)	 - Help for fileagent
* [`geneos floating`](geneos_floating.md)	 - Help for Floating Netprobes
* [`geneos gateway`](geneos_gateway.md)	 - Help for gateways
* [`geneos licd`](geneos_licd.md)	 - Help for Licence Daemon
* [`geneos netprobe`](geneos_netprobe.md)	 - Help for Netprobes
* [`geneos san`](geneos_san.md)	 - Help for Self-Announcing Netprobes
* [`geneos webserver`](geneos_webserver.md)	 - Help for Web Dashboard Servers

The `geneos` program will help you manage your Geneos environment.


With `geneos` you can initialise a new installation, install and
update software releases, add and remove instances, control processes
and build template based configuration files for SANs and more.


The subsystems group the management of related functions together.

Most commands 

### Options

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## Examples

```bash
geneos init demo -u jondoe@example.com -l
geneos ps
geneos restart

```

## SEE ALSO

