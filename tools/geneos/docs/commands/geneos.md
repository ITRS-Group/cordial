# `geneos`

Control your Geneos environment

## Commands

* [`geneos` add](`geneos`_add.md)	 - Add a new instance
* [`geneos` aes](`geneos`_aes.md)	 - Manage Geneos compatible key files and encode/decode passwords
* [`geneos` clean](`geneos`_clean.md)	 - Clean-up instance directories
* [`geneos` command](`geneos`_command.md)	 - Show command line and environment for instances
* [`geneos` config](`geneos`_config.md)	 - Configure geneos command environment
* [`geneos` copy](`geneos`_copy.md)	 - Copy instances
* [`geneos` delete](`geneos`_delete.md)	 - Delete instances
* [`geneos` disable](`geneos`_disable.md)	 - Disable instances
* [`geneos` enable](`geneos`_enable.md)	 - Enable instance
* [`geneos` home](`geneos`_home.md)	 - Output a directory path for given options
* [`geneos` host](`geneos`_host.md)	 - Manage remote host settings
* [`geneos` import](`geneos`_import.md)	 - Import files to an instance or a common directory
* [`geneos` init](`geneos`_init.md)	 - Initialise a Geneos installation
* [`geneos` login](`geneos`_login.md)	 - Store credentials for software downloads
* [`geneos` logout](`geneos`_logout.md)	 - Logout (remove credentials)
* [`geneos` logs](`geneos`_logs.md)	 - Show log(s) for instances
* [`geneos` ls](`geneos`_ls.md)	 - List instances
* [`geneos` migrate](`geneos`_migrate.md)	 - Migrate legacy .rc configuration to new formats
* [`geneos` move](`geneos`_move.md)	 - Move (or rename) instances
* [`geneos` package](`geneos`_package.md)	 - A brief description of your command
* [`geneos` protect](`geneos`_protect.md)	 - Mark instances as protected
* [`geneos` ps](`geneos`_ps.md)	 - List process information for instances, optionally in CSV or JSON format
* [`geneos` rebuild](`geneos`_rebuild.md)	 - Rebuild instance configuration files
* [`geneos` reload](`geneos`_reload.md)	 - Reload instance configuration, where supported
* [`geneos` restart](`geneos`_restart.md)	 - Restart instances
* [`geneos` revert](`geneos`_revert.md)	 - Revert migration of .rc files from backups
* [`geneos` set](`geneos`_set.md)	 - Set instance configuration parameters
* [`geneos` show](`geneos`_show.md)	 - Show runtime, global, user or instance configuration is JSON format
* [`geneos` snapshot](`geneos`_snapshot.md)	 - Capture a snapshot of each matching dataview
* [`geneos` start](`geneos`_start.md)	 - Start instances
* [`geneos` stop](`geneos`_stop.md)	 - Stop instances
* [`geneos` tls](`geneos`_tls.md)	 - Manage certificates for secure connections
* [`geneos` unset](`geneos`_unset.md)	 - Unset a configuration value
* [`geneos` version](`geneos`_version.md)	 - Show program version details

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

