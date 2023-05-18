# `geneos`

Control your Geneos environment

## Commands

* [`geneos add`](geneos_add.md)	 - Add a new instance
* [`geneos aes`](geneos_aes.md)	 - Manage Geneos compatible key files and encode/decode passwords
* [`geneos clean`](geneos_clean.md)	 - Clean-up instance directories
* [`geneos command`](geneos_command.md)	 - Show command line and environment for instances
* [`geneos config`](geneos_config.md)	 - Configure geneos command environment
* [`geneos copy`](geneos_copy.md)	 - Copy instances
* [`geneos delete`](geneos_delete.md)	 - Delete instances
* [`geneos disable`](geneos_disable.md)	 - Disable instances
* [`geneos enable`](geneos_enable.md)	 - Enable instance
* [`geneos home`](geneos_home.md)	 - Output a directory path for given options
* [`geneos host`](geneos_host.md)	 - Manage remote host settings
* [`geneos import`](geneos_import.md)	 - Import files to an instance or a common directory
* [`geneos init`](geneos_init.md)	 - Initialise a Geneos installation
* [`geneos login`](geneos_login.md)	 - Store credentials for software downloads
* [`geneos logout`](geneos_logout.md)	 - Logout (remove credentials)
* [`geneos logs`](geneos_logs.md)	 - Show log(s) for instances
* [`geneos ls`](geneos_ls.md)	 - List instances
* [`geneos migrate`](geneos_migrate.md)	 - Migrate legacy .rc configuration to new formats
* [`geneos move`](geneos_move.md)	 - Move (or rename) instances
* [`geneos package`](geneos_package.md)	 - A brief description of your command
* [`geneos protect`](geneos_protect.md)	 - Mark instances as protected
* [`geneos ps`](geneos_ps.md)	 - List process information for instances, optionally in CSV or JSON format
* [`geneos rebuild`](geneos_rebuild.md)	 - Rebuild instance configuration files
* [`geneos reload`](geneos_reload.md)	 - Reload instance configuration, where supported
* [`geneos restart`](geneos_restart.md)	 - Restart instances
* [`geneos revert`](geneos_revert.md)	 - Revert migration of .rc files from backups
* [`geneos set`](geneos_set.md)	 - Set instance configuration parameters
* [`geneos show`](geneos_show.md)	 - Show runtime, global, user or instance configuration is JSON format
* [`geneos snapshot`](geneos_snapshot.md)	 - Capture a snapshot of each matching dataview
* [`geneos start`](geneos_start.md)	 - Start instances
* [`geneos stop`](geneos_stop.md)	 - Stop instances
* [`geneos tls`](geneos_tls.md)	 - Manage certificates for secure connections
* [`geneos unset`](geneos_unset.md)	 - Unset a configuration value
* [`geneos version`](geneos_version.md)	 - Show program version details

## Details

Manage and control your Geneos environment. With `geneos` you can
initialise a new installation, add and remove components, control
processes and build template based configuration files for SANs and
more.

### Options

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## Examples

```bash
$ geneos start
$ geneos ps

```

## SEE ALSO

