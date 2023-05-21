# `geneos init floating`

Initialise a Geneos Floating Netprobe environment

```text
geneos init floating [flags] [USERNAME] [DIRECTORY]
```

Install a Floating Netprobe into a new Geneos install
directory.

Without any flags the command installs a Floating Netprobe in a directory called
`geneos` under the user's home directory (unless the user's home
directory ends in `geneos` in which case it uses that directly),
downloads the latest netprobe release and create a netprobe instance using
the `hostname` of the system.

In almost all cases authentication will be required to download the
Netprobe package and as this is a new Geneos installation it is
unlikely that the download credentials are saved in a local config
file, so use the `-u email@example.com` as appropriate.

If you have a netprobe software archive locally then use the `-A
PATH`. If the name of the file is not in the same format as
downloaded from the official site(s) then you have to also set the
type (netprobe) and version using the `-T [TYPE:]VERSION`. TYPE is
set to `netprobe` if not given. 

The initial configuration file is built from the default templates
installed and located in `.../templates` but this can be overridden
with the `-s` option. You can set `gateways`, `types`, `attributes`,
`variables` using the appropriate flags. These flags can be specified
multiple times.

### Options

```text
  -V, --version VERSION              Download this VERSION, defaults to latest. Doesn't work for EL8 archives. (default "latest")
  -A, --archive string               Directory of releases for installation
  -T, --override [TYPE:]VERSION      Override the [TYPE:]VERSION for archive files with non-standard names
  -g, --gateway HOSTNAME:PORT        A gateway connection in the format HOSTNAME:PORT
                                     (Repeat as required, san and floating only)
  -a, --attribute NAME=VALUE         An attribute in the format NAME=VALUE
                                     (Repeat as required, san only)
  -t, --type NAME                    A type NAME
                                     (Repeat as required, san only)
  -v, --variable [TYPE:]NAME=VALUE   A variable in the format [TYPE:]NAME=VALUE
                                     (Repeat as required, san only)
```

## SEE ALSO

* [geneos init](geneos_init.md)	 - Initialise The Installation
