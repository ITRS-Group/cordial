# `geneos init demo`

The `geneos init demo` command is used to initialise a Geneos Demo environment.

Without any arguments the command initialises a directory called `geneos` under your user's home directory (unless the user's home directory ends in `geneos` in which case it uses that directly), downloads the latest release archives and creates a Gateway instance using the name `Demo Gateway` (with embedded space) as required for Demo licensing, as Netprobe and a Webserver.

If given the `--minimal`/`-M` flag then the minimal Netprobe component is deployed, which can save 300MB+ of download when being run, for example, to build a docker container.

If the release archive files required have already been downloaded then use the `--archive`/`-A` flag to indicate their location. For each component type this directory is checked for the latest release.

To fetch the releases from the ITRS download server, authentication will be required. Use the `--username`/`-u` flag to specify the user account and you will be prompted for a password.

The initial configuration file for the Gateway is built from the default templates, which are installed at the same time, and located in `gateway/templates` but this can be overridden with the `-s` option. For the Gateway component you can add include files using `-i PRIORITY:PATH` flag. This can be repeated multiple times.

Other flags inherited from the `geneos init` command can be used to control the installation.

If you are running on a Linux system that is capable of supporting the X Window System (which is anything with a desktop display, including WSL2) and you have an environment variable `DISPLAY` set, then the command will also install an Active Console for Linux. This is not started by default. This does not currently work from a docker container.

## Example Usage

You can set-up a Demo environment like this:

```bash
geneos init demo -u email@example.com
```

or, to script this, do:

```bash
export ITRS_DOWNLOAD_USERNAME=email@example.com
export ITRS_DOWNLOAD_PASSWORD=mysecret
geneos init demo
```

Here you should replace the email address with the one you registered on the ITRS Resources website and the command will prompt you for your password if not supplied. The ITRS_DOWNLOAD_PASSWORD environment variable can also use "expandable format" which is the output of the `geneos aes password` command.

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
