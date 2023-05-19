# `geneos login`

Store credentials related to Geneos

```text
geneos login [flags] [DOMAIN]
```

The login command will store credentials in your local configuration
directory for use with `geneos` and other tools from `cordial`.
Passwords are encrypted using a key file which is created if it does not
already exist.

If not given `DOMAIN` defaults to `itrsgroup.com`. When credentials are
used, the destination is checked against all stored credentials and the
longest match is selected.

A common use of stored credentials is for the download and installation
of Geneos packages via the `geneos package` subsystem. Credentials are
also used by the `geneos snapshot` command and the `dv2email` program.

If no `--username`/`-u` option is given then the user is prompted for
one.

If no `--password`/`-p` is given then the user is prompted to enter the
password twice and it is only accepted if both instances match. After
three failures to match password the program will terminate and not save
the credential.

The user's default key file is used unless the `--keyfile`/`-k` is
given. The path to the key file used is stored in the credential and so
if the key file is moved or overwritten then that credential becomes
unusable.

The credentials cannot be used without the original key file and each
set of credentials can use a separate key file.

The credentials file itself can be world readable as the security is
through the use of a protected key file. Running `geneos.exe` on Windows
does not currently protect the key file on creation.

Future releases will support extended credential sets, for example SSH
and 2-legged OAuth ClientID/ClientSecret (such as application keys from
cloud providers). Another addition may be the automatic encryption of
non-password data in credentials.

### Options

```text
  -u, --username string      Username
  -p, --password PLAINTEXT   Password
  -k, --keyfile KEYFILE      Key file to use
  -l, --list                 List the names of the currently stored credentials
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
