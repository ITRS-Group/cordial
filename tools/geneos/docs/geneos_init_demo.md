# `geneos init demo`

Initialise a Geneos Demo environment, creating a new directory structure as required.

Without any flags the command installs the components in a directory called `geneos` under the user's home directory (unless the user's home directory ends in `geneos` in which case it uses that directly), downloads the latest release archives and creates a Gateway instance using the name `Demo Gateway` (with embedded space) as required for Demo licensing, as Netprobe and a Webserver.

If given the `--minimal`/`-M` flag then the minimal Netprobe component is deployed, which can save 300MB+ of download when being run, for example, to build a docker container.

If the release archive files required have already been downloaded then use the `-A directory` flag to indicate their location. For each component type this directory is checked for the latest release.

Otherwise, to fetch the releases from the ITRS download server authentication will be required use the `-u email@example.com` to specify the user account and you will be prompted for a password.

The initial configuration file for the Gateway is built from the default templates installed and located in `.../templates` but this can be overridden with the `-s` option. For the Gateway you can add include files using `-i PRIORITY:PATH` flag. This can be repeated multiple times.

Other flags inherited from the `geneos init` command can be used to influence the installation.

## Usage

```text
geneos init demo [flags] [USERNAME] [DIRECTORY]
```

### Options

```text
  -M, --minimal                          use a minimal Netprobe release
  -i, --include PRIORITY:[PATH|URL]      An include file in the format PRIORITY:[PATH|URL]
                                         (Repeat as required, gateway only)
  -l, --log                              Follow logs after starting instance(s)
  -F, --force                            Ignore existing directories and files and overwrite
  -n, --name string                      Use name for instances and configurations instead of the hostname
  -T, --tls                              Create internal certificates for TLS support
  -C, --signing-bundle string            signing bundle in PEM or PFX/PKCS#12 format.
                                         Use a dash ('-') to be prompted for PEM from console.
                                         PFX/PKCS#12 must be files and are identified by the .pfx or .p12 file extension
      --signing-bundle-password SECRET   Password for PFX/PKCS#12 certificate file.
                                         You will be prompted if required and not supplied as an argument.
      --insecure                         Do not create internal certificates for TLS support
  -A, --archive string                   Directory of releases for installation
  -N, --nexus                            Download from nexus.itrsgroup.com. Requires ITRS internal credentials
  -S, --snapshots                        Download from nexus snapshots. Requires -N
  -V, --version VERSION                  Download matching VERSION, defaults to latest. Doesn't work for EL8 archives. (default "latest")
  -u, --username string                  Username for downloads (password prompted)
  -w, --gateway-template string          A gateway template file
  -e, --env NAME=VALUE                   Environment variable for instance start-up
                                         (Repeat as required)
      --header NAME=VALUE                HTTP header in the format NAME=VALUE
                                         (Repeat as required)
      --allow-root                       allow running as root (not recommended)
  -G, --config string                    config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME                    Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos init](geneos_init.md)	 - Initialise The Installation
