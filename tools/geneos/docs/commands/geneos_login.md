# `geneos login`

Store credentials for software downloads

```text
geneos login [flags] [URLPATTERN]
```

## Details

Prompt for and stored credentials for later use by commands.

Typical use is for downloading release archives from the official ITRS web
site.

If not given `URLPATTERN` defaults to `itrsgroup.com`. When credentials are
used, the destination is checked against all stored credentials and the
longest match is selected.

If no `-u USERNAME` is given then the user is prompted for a username.

If no `-p PASSWORD` is given then the user is prompted for the password,
which is not echoed, twice and it is only accepted if both instances match.

The credentials are encrypted with the keyfile specified with `-k KEYFILE`
and if not given then the user's default keyfile is used - and created if it
does not exist. See `geneos aes new` for details.

The credentials cannot be used without the keyfile and each set of
credentials can use a separate keyfile.

### Options

```text
  -u, --username string      Username
  -p, --password PLAINTEXT   Password
  -k, --keyfile KEYFILE      Keyfile to use
  -l, --list                 list domains of credentials (no validity checks are done)
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
