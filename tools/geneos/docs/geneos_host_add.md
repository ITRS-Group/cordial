# `geneos host add`

Add a remote host `NAME` for seamless control of your Geneos estate.

One or both of `NAME` or `SSHURL` must be given. `NAME` is used as the default hostname if not `SSHURL` is given and, conversely, the hostname portion of the `SSHURL` is used if no NAME is supplied.

The `SSHURL` extends the standard format by allowing a path to the root directory for the remote Geneos installation in the format:

  ssh://[USER@]HOST[:PORT][/PATH]

Here:

`USER` is the username to be used to connect to the target host. If is not defined, it will default to the current username.

`PORT` is the ssh port used to connect to the target host. If not defined the default is 22.

`HOST` the hostname or IP address of the target host. Required.
  
`PATH` is the root Geneos directory used on the target host. If not defined, it is set to the same as the local Geneos root directory.

```text
geneos host add [flags] [NAME] [SSHURL]
```

### Options

```text
  -I, --init                 Initialise the remote host directories and component files
  -p, --prompt               Prompt for password
  -P, --password PLAINTEXT   Password
  -k, --keyfile KEYFILE      Keyfile for encryption of stored password (default /home/peter/.config/docs/keyfile.aes)
  -i, --privatekey PATH      Private key file
```

## Examples

```bash
geneos host add server1
geneos host add ssh://server2:50122
geneos host add remote1 ssh://server.example.com/opt/geneos

```

## SEE ALSO

* [geneos host](geneos_host.md)	 - Remote Host Operations
