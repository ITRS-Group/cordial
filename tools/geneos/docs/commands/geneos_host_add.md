## geneos host add

Add a remote host

### Synopsis


Add a remote host `NAME` for seamless control of your Geneos estate.

One or both of `NAME` or `SSHURL` must be given. `NAME` is used as
the default hostname if not `SSHURL` is given and, conversely, the
hostname portion of the `SSHURL` is used if no NAME is supplied.

The `SSHURL` extends the standard format by allowing a path to the
root directory for the remote Geneos installation in the format:

  ssh://[USER@]HOST[:PORT][/PATH]

Here:

`USER` is the username to be used to connect to the target host. If
is not defined, it will default to the current username.

`PORT` is the ssh port used to connect to the target host. If not
defined the default is 22.

`HOST` the hostname or IP address of the target host. Required.
  
`PATH` is the root Geneos directory used on the target host. If not
defined, it is set to the same as the local Geneos root directory.


```
geneos host add [flags] [NAME] [SSHURL]
```

### Examples

```

geneos host add server1
geneos host add ssh://server2:50122
geneos host add remote1 ssh://server.example.com/opt/geneos

```

### Options

```
  -I, --init                 Initialise the remote host directories and component files
  -p, --prompt               Prompt for password
  -P, --password PLAINTEXT   Password
  -k, --keyfile KEYFILE      Keyfile (default /home/peter/.config/geneos/keyfile.aes)
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

### SEE ALSO

* [geneos host](geneos_host.md)	 - Manage remote host settings

