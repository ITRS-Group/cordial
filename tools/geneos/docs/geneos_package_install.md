# `geneos package install`

The `install` command will, with no other options, download and unarchive the latest releases of each supported Geneos component from the official ITRS download servers.

💡 Downloads require an ITRS client login and credentials must be provided.

Download credentials can be from those stored locally by running the `geneos login` command before `geneos package install` or they supplied on the command line with the --`username`|`-u` option, which will prompt for a password. In previous versions of cordial you may have also stored credentials in the `geneos` program's own configuration, which will still be used if available.

The deprecated `--pwfile`/`-P` option can be used to refer to a local file containing the password.

If you have already downloaded the release archives then you can use the `--local`/`-L` option to use local files. If you do not supply a file name or directory on the command line then the command will look in the `packages/downloads` directory under the Geneos installation directory.

Downloads are normally saved in the above directory but this can be disabled with the `--nosave`/`-n` option. This is the default if you install locally from a specific directory or file too.

With the `--update`/`-U` option the command will also update the active versions for base specified by `--base`/`-b` (default of `active_prod`) by stopping any instances that use that base name and starting them again after updating the links. Because links are potentially shared by many instances the install may succeed but the update fail if any instances are protected. To also update protected instances use the `--force`/`-F` flag. Note that all matching instances will be stopped, even those that may not be updated, as it is not wholly predictable what version may be installed and the instances must be stopped beforehand.

The `--force`/`-F` flags implies `--update`.

By default the latest version found will be the one installed, either from the download site or locally. To install a specific version from the use the `--version`/`-V` option with a version in the form `MAJOR.MINOR.PATCH` where omitting `PATCH` will get the latest patch release for `MAJOR.MINOR` and omitting `MINOR.PATCH` will get the latest version in the `MAJOR` set. Versions cannot be selected for remote `el8` archives because of a restriction in indexing releases. Specifying a version with either a local only or with a directory name on the command line will apply the same rules to all matching local files.

If you have downloaded a release but have changed the file name format then you must use the `--override`/`-O` option to tell the `install` command what component type and release the archive contains, e.g. `-T gateway:5.12.1`. The command will not validate your option and will simply unarchive the file, if it can, in the directory that would be created for that component and version.

For internal ITRS users there are the `--nexus`/`-N` and `--snapshot`/`-S` options to download archives from the internal nexus server. The `--snapshot`/`-S` option implies `--nexus`/`-N`. You may need to supply different credentials for these downloads.

Installations can be limited to a specific host with the global `--host`/`-H` option otherwise the installation is done to all configured hosts.

Finally, if you just want to download releases and not install them - so you can put them on a shred drive for example - then you can use the `--download`/`-D` option. This will download the selected releases to the current directory, or if you give a directory on the command line then to that directory. Note that this option makes no sense with a number of other command line options and will error if those are given, e.g. `--local/-L` and so on.

To use a proxy, when direct connectivity from your server may not be available, set the appropriate environment variables as detailed in the Go documentation: <https://pkg.go.dev/net/http#ProxyFromEnvironment>. The values of these variables are the same as for the industry-standard examples you will find on the web.

```text
geneos package install [flags] [TYPE] [FILE|URL...]
```

### Options

```text
  -u, --username string   Username for downloads, defaults to configuration value in download.username
  -L, --local             Install from local files only
  -n, --nosave            Do not save a local copy of any downloads
  -D, --download          Download only
  -U, --update            Update the base directory symlink, will restart unprotected instances
  -F, --force             Force restart of protected instances, implies --update
  -b, --base string       Override the base active_prod link name (default "active_prod")
  -V, --version string    Download this version, defaults to latest. Doesn't work for EL8 archives. (default "latest")
  -O, --override string   Override the TYPE:VERSION for archive files with non-standard names
  -N, --nexus             Download from nexus.itrsgroup.com. Requires auth.
  -S, --snapshots         Download from nexus snapshots (pre-releases), not releases. Requires -N
```

## Examples

```bash
geneos install gateway
geneos install fa2 5.5 -U
geneos install netprobe -b active_dev -U

```

## SEE ALSO

* [geneos package](geneos_package.md)	 - Package Operations
