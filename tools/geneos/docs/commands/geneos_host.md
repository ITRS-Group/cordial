# `geneos host`

Manage remote host settings

```text
geneos host
```
## Commands

* [`geneos host add`](geneos_host_add.md)	 - Add a remote host
* [`geneos host delete`](geneos_host_delete.md)	 - Delete a remote host configuration
* [`geneos host list`](geneos_host_list.md)	 - List hosts, optionally in CSV or JSON format
* [`geneos host set`](geneos_host_set.md)	 - Set host configuration value
* [`geneos host show`](geneos_host_show.md)	 - Show details of remote host configuration

## Details

# `geneos host` Subsystem

The host subsystem manages all the tasks and configuration related to
remote hosts.

Each host you create maps remote Geneos installations to local commands.
For almost any command that accepts instance names you can qualify them
with an '@host' suffix to indicate which remote host that instance is on.

Currently only SSH is supported.



## Remote Management

The `geneos` command can transparently manage instances across multiple
systems using SSH.

### What does this mean?

See if these commands give you a hint:

```bash
geneos host add server2 ssh://geneos@myotherserver.example.com/opt/geneos
geneos add gateway newgateway@server2
geneos start
```

Command like `ls` and `ps` will works transparently and merge all
instances together, showing you where they are configured to run.

The format of the SSH URL has been extended to include the Geneos
directory and for the `add host` command is:

`ssh://[USER@]HOST[:PORT][/PATH]`

If not set, USER defaults to the current username. Similarly PORT
defaults to 22. PATH defaults to the local Geneos path. The most basic
SSH URL of the form `ssh://hostname` results in a remote accessed as the
current user on the default SSH port and rooted in the same directory as
the local set-up. Is the remote directory is empty (dot files are
ignored) then the standard file layout is created. If you do not provide
any SSH URL then the hostname is taken from the name of the host - e.g.

```bash
geneos host add myserver
```

is taken as:

```bash
geneos host add myserver ssh://myserver
```

### How does it work?

There are a number of prerequisites for remote support:

1. Remote hosts must be Linux on amd64

2. Password-less SSH access, either via an `ssh-agent` or unprotected
   private keys

3. At this time the only private keys supported are those in your `.ssh`
   directory beginning `id_` - later updates will allow you to set the
   name of the key to load, but using an agent is recommended.

4. The remote user must be configured to use a `bash` shell or similar.
   See limitations below.

If you can log in to a remote Linux server using `ssh user@server` and
not be prompted for a password or passphrase then you are set to go.
It's beyond the scope of this README to explain how to set-up
`ssh-agent` or how to create an unprotected private key file, so please
search online.

### Limitations

The remote connections over SSH mean there are limitations to the
features available on remote servers:

1. Control over instance processes is done via shell commands and little
   error checking is done, so it is possible to cause damage and/or
   processes not to to start or stop as expected.

2. All actions are taken as the user given in the SSH URL (which should
   NEVER be `root`) and so instances that are meant to run as other
   users cannot be controlled. Files and directories may not be
   available if the user does not have suitable permissions.

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
