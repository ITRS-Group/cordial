# `geneos init all`

Initialise a more complete Geneos environment

```text
geneos init all [flags] [USERNAME] [DIRECTORY]
```

Initialise a typical Geneos installation.

This command initialises a Geneos installation by:

* Creating the directory structure & user configuration file,
* Installing software packages for component types `gateway`, `licd`, `netprobe` & `webserver`,
* Creating an instance for each component type named after the hostname (except for `netprobe` whose instance is named `localhost`)
* Starting the created instances.

A license file is required and should be given using option `-L`. If a license file is not available, then use `-L /dev/null` which will create an empty `geneos.lc` file that can be overwritten later.

Authentication will most-likely be required to download the installation software packages and, as this is a new Geneos installation, it is unlikely that the download credentials are saved in a local config file. Use option `-u email@example.com` to define the username for downloading software packages.

If packages are already downloaded locally, use option `-A Path_To_Archive` to refer to the directory containing the package archives.  Package files must be named in the same format as those downloaded from the [ITRS download portal](https://resources.itrsgroup.com/downloads). If no version is given using option `-V`, then the latest version of each component is installed.
### Options

```text
  -L, --licence string                Licence file location (default "geneos.lic")
  -A, --archive string                Directory of releases for installation
  -i, --include PRIORITY:{URL|PATH}   A gateway connection in the format HOSTNAME:PORT
                                      (Repeat as required, san and floating only)
```

## Examples

```bash
geneos init all -L https://myserver/files/geneos.lic -u email@example.com
geneos init all -L ~/geneos.lic -A ~/downloads /opt/itrs
sudo geneos init all -L /tmp/geneos-1.lic -u email@example.com myuser /opt/geneos

```

## SEE ALSO

* [geneos init](geneos_init.md)	 - Initialise The Installation
