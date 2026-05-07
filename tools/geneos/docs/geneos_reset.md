# `geneos reset`

The `reset` command resets the matching instance directories to a near-default state. The instances are stopped, the directories cleaned and the instance restarted. Because the command can have side-effects, to match all instances you must provide the `all` keyword as `geneos reset` will not match any instances by default.

The files and directrories that are removed are based on the component type and the global settings using the `clean` and `purge` pattern lists.
## Usage

```text
geneos reset [flags] [TYPE] [NAME...]
```

### Options

```text
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## Examples

```bash
# Stop all netprobes and remove all non-essential files from working 
# directories, then restart netprobes
geneos reset netprobe

```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
