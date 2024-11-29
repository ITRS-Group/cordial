# `geneos`

The `geneos` program will help you manage your Geneos environment.

The program will help you initialise a new installation, migrate an old `geneos-utils` one, install and update software releases, add and remove instances, control processes and build template based configuration files for SANs and more.

The program works best on Linux but is built for both Windows and MacOS. In the latter two cases it is primarily for the remote management (see below) of Linux instances. There is no support at this time for managing local Windows or MacOS instances of Geneos components.

Most commands work on "instances" of "components". As the names suggest, a "component" is a type of Geneos component such as a Gateway or a Netprobe. An "instance" is a configured instance of a component, a specific Gateway and so on. For a list of components see the "Registered Component Types" below. Each component has it's own help which you can read using `geneos gateway help` etc.

Instances can be created locally or on remote hosts over SSH connections without the need to install `geneos` on the remote Linux server.

Instance names are in the form `[TYPE]:NAME[@HOST]`, where the `[...]` mean that part is optional. The `TYPE` is only used to select the underlying type of Netprobe, e.g. Fix Analyser or plain, for Self-Announcing and Floating Netprobe components during deployment. The `HOST` part is the name of a configured remote host (which may not be the hostname); see the `host` sub-system help with `geneos host help` for more information.

For many commands you can also use wildcards for the `NAME` part. These wildcards are not complex regular expressions but instead follow more common file system patterns. (Note that the exact patterns support are the same as for the Go [`path.Match`](https://pkg.go.dev/path#Match) function.). These wildcards only work on `NAME` and not the `HOST` part and then only for those commands where they make sense, such as `geneos ls`, `geneos start` and so on.

The subsystems below group related functions together and have their own sub-commands, such as `geneos aes password` and `geneos init demo`. Use `geneos SUBSYSTEM help` to see more or, if you are reading this online you should be able to click through for further information.

## Subsystems

| Command | Description |
|-------|-------|
| [`geneos aes`](geneos_aes.md)	 | AES256 Key File Operations |
| [`geneos config`](geneos_config.md)	 | Configure Command Behaviour |
| [`geneos host`](geneos_host.md)	 | Remote Host Operations |
| [`geneos init`](geneos_init.md)	 | Initialise The Installation |
| [`geneos package`](geneos_package.md)	 | Package Operations |
| [`geneos tls`](geneos_tls.md)	 | TLS Certificate Operations |

---

## Control Instances

| Command / Aliases | Description |
|-------|-------|
| [`geneos reload / refresh`](geneos_reload.md)	 | Reload Instance Configurations |
| [`geneos restart`](geneos_restart.md)	 | Restart Instances |
| [`geneos start`](geneos_start.md)	 | Start Instances |
| [`geneos stop`](geneos_stop.md)	 | Stop Instances |

---

## Inspect Instances

| Command / Aliases | Description |
|-------|-------|
| [`geneos command`](geneos_command.md)	 | Show Instance Start-up Details |
| [`geneos home`](geneos_home.md)	 | Display Instance and Component Home Directories |
| [`geneos list / ls`](geneos_list.md)	 | List Instances |
| [`geneos logs / log`](geneos_logs.md)	 | View Instance Logs |
| [`geneos ps / status`](geneos_ps.md)	 | List Running Instance Details |
| [`geneos show / details`](geneos_show.md)	 | Show Instance Configuration |

---

## Manage Instances

| Command / Aliases | Description |
|-------|-------|
| [`geneos clean`](geneos_clean.md)	 | Clean-up Instance Directories |
| [`geneos copy / cp`](geneos_copy.md)	 | Copy instances |
| [`geneos disable`](geneos_disable.md)	 | Disable instances |
| [`geneos enable`](geneos_enable.md)	 | Enable instance |
| [`geneos move / mv / rename`](geneos_move.md)	 | Move instances |
| [`geneos protect`](geneos_protect.md)	 | Mark instances as protected |

---

## Configure Instances

| Command / Aliases | Description |
|-------|-------|
| [`geneos add`](geneos_add.md)	 | Add a new instance |
| [`geneos delete / rm`](geneos_delete.md)	 | Delete Instances |
| [`geneos deploy`](geneos_deploy.md)	 | Deploy a new Geneos instance |
| [`geneos import`](geneos_import.md)	 | Import Files To Instances Or Components |
| [`geneos migrate`](geneos_migrate.md)	 | Migrate Instance Configurations |
| [`geneos rebuild`](geneos_rebuild.md)	 | Rebuild Instance Configurations From Templates |
| [`geneos revert`](geneos_revert.md)	 | Revert Migrated Instance Configuration |
| [`geneos set`](geneos_set.md)	 | Set Instance Parameters |
| [`geneos unset`](geneos_unset.md)	 | Unset Instance Parameters |

---

## Manage Credentials

| Command | Description |
|-------|-------|
| [`geneos login`](geneos_login.md)	 | Enter Credentials |
| [`geneos logout`](geneos_logout.md)	 | Remove Credentials |

---

## Miscellaneous

| Command | Description |
|-------|-------|
| [`geneos snapshot`](geneos_snapshot.md)	 | Capture a snapshot of each matching dataview |
| [`geneos version`](geneos_version.md)	 | Show program version |

---

## Component Types

| Command | Description |
|-------|-------|
| [`geneos ca3`](geneos_ca3.md)	 | Collection Agent 3 |
| [`geneos fa2`](geneos_fa2.md)	 | Fix Analyser 2 |
| [`geneos fileagent`](geneos_fileagent.md)	 | File Agent |
| [`geneos floating`](geneos_floating.md)	 | Floating Netprobes |
| [`geneos gateway`](geneos_gateway.md)	 | Gateways |
| [`geneos licd`](geneos_licd.md)	 | Licence Daemon |
| [`geneos minimal`](geneos_minimal.md)	 | Minimal Netprobes |
| [`geneos netprobe`](geneos_netprobe.md)	 | Netprobes |
| [`geneos san`](geneos_san.md)	 | Self-Announcing Netprobes |
| [`geneos webserver`](geneos_webserver.md)	 | Web Dashboard Servers |

### Options

```text
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## Examples

```bash
geneos init demo -u email@example.com -l
geneos ps
geneos restart

```

## SEE ALSO

