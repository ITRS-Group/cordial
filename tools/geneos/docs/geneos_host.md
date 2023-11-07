# `geneos host`

The `geneos` program can seamlessly manage Geneos instances across multiple servers. By transparently and securely connecting to other Linux systems running Geneos components you can do all the same things as with other Geneos instance that you have created locally.

This can be as simple at:

```bash
geneos add host myServer2
geneos add gateway Gateway2@myServer2
geneos start -H myServer2
```

Almost any command that accepts names to match instances can handle names like `name@host`.

Commands like `ls` and `ps` will works transparently and merge all instances together, showing you where they are configured to run.

The host name, as opposed to _hostname_, that you use when you add a host to `geneos` does not have to be the same as the underlying _hostname_ of the remote server.

## Adding Hosts

Currently only SSH is supported.

The recommended way to ensure secure connectivity is to use and agent with SSH keys. If this is not available then you can also use locally stored, unprotected (as in without a passphrase) private keys and finally locally stored encrypted passwords.

Future releases may add support for protected private keys (through the local encrypted storage of passphrases) or for Kerberos (GSS-API) authentication. An SSH agent will still remain the recommended method.

Please note that any configuration in your `ssh_config` file(s) is ignored. Also, you must have already added the remote SSH server key to your `known_hosts` file or connections will fail with an error.

When you add a host you must use either a local label (`HOST` in the examples and documentation) and/or an SSH URL that refers to the connection details and the remote location of the Geneos installation. The `HOST` is taken from the label, if given, or else the `HOSTNAME` part of the SSH URL.

The standard SSH URL has been extended to support the Geneos installation directory and for the `add host` command it looks like:

`ssh://[USER@]HOSTNAME[:PORT][/PATH]`

If not set, `USER` defaults to your current username. Similarly `PORT` defaults to 22. `PATH` defaults to the local Geneos installation path. The most basic SSH URL of the form `ssh://hostname` results in access as the current user on the default SSH port and rooted in the same directory as the local set-up. Is the remote directory is empty (dot files are ignored) then the standard file layout is created.

`HOSTNAME` can be an IP address, as can the `HOST` label but this is not recommended.

```bash
geneos host add server1 ssh://myserver.example.com
```

### Prerequisites

There are other prerequisites for remote support:

* Remote hosts must be Linux

* The remote user must be configured to use a `bash` shell or similar. See limitations below.

If you can log in to a remote Linux server using `ssh user@server` and not be prompted for a password or passphrase then you are set to go. It's beyond the scope of this README to explain how to set-up `ssh-agent` or how to create an unprotected private key file, so please take a look online.

### Limitations

The remote connections over SSH mean there are limitations to the features available on remote servers:

* Control over instance processes is done via shell commands and little error checking is done, so it is possible to cause damage and/or processes not to to start or stop as expected.

* All actions are taken as the user given in the SSH URL (which should NEVER be `root`) and so instances that are meant to run as other users cannot be controlled. Files and directories may not be available if the user does not have suitable permissions.


## Commands

| Command | Description |
|-------|-------|
| [`geneos host add`](geneos_host_add.md)	 | Add a remote host |
| [`geneos host delete`](geneos_host_delete.md)	 | Delete a remote host configuration |
| [`geneos host list`](geneos_host_list.md)	 | List hosts, optionally in CSV or JSON format |
| [`geneos host set`](geneos_host_set.md)	 | Set host configuration value |
| [`geneos host show`](geneos_host_show.md)	 | Show details of remote host configuration |

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
