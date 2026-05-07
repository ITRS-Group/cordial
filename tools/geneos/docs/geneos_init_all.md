# `geneos init all`

Initialise a typical Geneos installation.

This command initialises a Geneos installation by:

* Creating the directory structure & user configuration file,
* Installing software packages for component types `gateway`, `licd`, `netprobe` & `webserver`,
* Creating an instance for each component type named after the hostname (except for `netprobe` whose instance is named `localhost`)
* Starting the created instances.

A license file is required and should be given using option `-L`. If a license file is not available, then use `-L /dev/null` which will create an empty `geneos.lc` file that can be overwritten later.

Authentication will most-likely be required to download the installation software packages and, as this is a new Geneos installation, it is unlikely that the download credentials are saved in a local config file. Use option `-u email@example.com` to define the username for downloading software packages.

If packages are already downloaded locally, use option `-A Path_To_Archive` to refer to the directory containing the package archives.  Package files must be named in the same format as those downloaded from the [ITRS download portal](https://resources.itrsgroup.com/downloads). If no version is given using option `-V`, then the latest version of each component is installed.
## Usage

```text
geneos init all [flags] [USERNAME] [DIRECTORY]
```

### Options

```text
  -L, --licence string                Licence file location (default "geneos.lic")
  -M, --minimal                       use a minimal Netprobe release
  -i, --include PRIORITY:{URL|PATH}   A gateway connection in the format HOSTNAME:PORT
                                      (Repeat as required, san and floating only)
  -l, --log                           Follow logs after starting instance(s)
  -F, --force                         Ignore existing directories and files and overwrite
  -n, --name string                   Use name for instances and configurations instead of the hostname
  -T, --tls                           Create internal certificates for TLS support
  -C, --signing-bundle string         signing bundle in PEM format.
                                      This bundle must contain an unencrypted private key
                                      and matching signing certificate and other certificates up to the root CA.
      --insecure                      Do not create internal certificates for TLS support
  -A, --archive string                Directory of releases for installation
  -N, --nexus                         Download from nexus.itrsgroup.com. Requires ITRS internal credentials
  -S, --snapshots                     Download from nexus snapshots. Requires -N
  -V, --version VERSION               Download matching VERSION, defaults to latest. Doesn't work for EL8 archives. (default "latest")
  -u, --username string               Username for downloads (password prompted)
  -w, --gateway-template string       A gateway template file
  -e, --env NAME=VALUE                Environment variable for instance start-up
                                      (Repeat as required)
      --header NAME=VALUE             HTTP header in the format NAME=VALUE
                                      (Repeat as required)
      --allow-root                    allow running as root (not recommended)
  -G, --config string                 config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME                 Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## Examples

```bash
geneos init all -L https://myserver/files/geneos.lic -u email@example.com
geneos init all -L ~/geneos.lic -A ~/downloads /opt/itrs
sudo geneos init all -L /tmp/geneos-1.lic -u email@example.com myuser /opt/geneos

```

## SEE ALSO

* [geneos init](geneos_init.md)	 - Initialise The Installation
