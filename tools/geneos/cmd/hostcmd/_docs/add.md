Add a remote host `NAME` for seamless control of your Geneos estate.

One or both of `NAME` or `SSHURL` must be given. `NAME` is used as the default hostname if not `SSHURL` is given and, conversely, the hostname portion of the `SSHURL` is used if no NAME is supplied.

The `SSHURL` extends the standard format by allowing a path to the root directory for the remote Geneos installation in the format:

  ssh://[USER@]HOST[:PORT][/PATH]

Here:

`USER` is the username to be used to connect to the target host. If is not defined, it will default to the current username.

`PORT` is the ssh port used to connect to the target host. If not defined the default is 22.

`HOST` the hostname or IP address of the target host. Required.
  
`PATH` is the root Geneos directory used on the target host. If not defined, it is set to the same as the local Geneos root directory.
