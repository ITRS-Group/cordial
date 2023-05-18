# `geneos host`

Manage remote host settings

```text
geneos host [flags]
```
## Commands

* [`geneos host add`](geneos_host_add.md)	 - Add a remote host
* [`geneos host delete`](geneos_host_delete.md)	 - Delete a remote host configuration
* [`geneos host ls`](geneos_host_ls.md)	 - List hosts, optionally in CSV or JSON format
* [`geneos host set`](geneos_host_set.md)	 - Set host configuration value
* [`geneos host show`](geneos_host_show.md)	 - Show details of remote host configuration

## Details

Manage remote host settings. Without a subcommand defaults to `ls` of hosts.

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
