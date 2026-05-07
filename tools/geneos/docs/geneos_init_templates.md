# `geneos init templates`

The `geneos` commands contains embedded template files that are normally written out during initialization of a new installation so that they can be customised if required. In the case of adopting a legacy installation or upgrading the program you should run this command to write-out the current default templates.

This command will overwrite any files with the same name but will not delete other template files that may already exist.

Use this command if you get missing template errors using the `add` command.

## Usage

```text
geneos init templates [flags]
```

### Options

```text
      --allow-root                allow running as root (not recommended)
  -A, --archive string            Directory of releases for installation
  -G, --config string             config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -e, --env NAME=VALUE            Environment variable for instance start-up
                                  (Repeat as required)
  -F, --force                     Ignore existing directories and files and overwrite
  -w, --gateway-template string   A gateway template file
      --header NAME=VALUE         HTTP header in the format NAME=VALUE
                                  (Repeat as required)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --insecure                  Do not create internal certificates for TLS support
  -l, --log                       Follow logs after starting instance(s)
  -n, --name string               Use name for instances and configurations instead of the hostname
  -N, --nexus                     Download from nexus.itrsgroup.com. Requires ITRS internal credentials
  -C, --signing-bundle string     signing bundle in PEM format.
                                  This bundle must contain an unencrypted private key
                                  and matching signing certificate and other certificates up to the root CA.
  -S, --snapshots                 Download from nexus snapshots. Requires -N
  -T, --tls                       Create internal certificates for TLS support
  -u, --username string           Username for downloads (password prompted)
  -V, --version VERSION           Download matching VERSION, defaults to latest. Doesn't work for EL8 archives. (default "latest")
```

## SEE ALSO

* [geneos init](geneos_init.md)	 - Initialise The Installation
