## geneos install

Install (remote or local) Geneos packages

### Synopsis


Installs files from FILE(s) in to the packages/ directory. The filename(s) must of of the form:

	`geneos-TYPE-VERSION*.tar.gz`

The directory for the package is created using the VERSION from the archive
filename unless overridden by the `-T` and `-V` flags.

If a TYPE is given then the latest version from the packages/downloads
directory for that TYPE is installed, otherwise it is treated as a
normal file path. This is primarily for installing to remote locations.

Install only changes a base link if one does not exist. To update an
existing base link use the `-U` option. The `-U` options stops any instance,
updates the link and starts the instance up again.

Use the update command to explicitly change the base link after installation.

Use the `-b` flag to change the base link name from the default `active_prod`. This also
applies when using `-U`.


```
geneos install [flags] [TYPE] | FILE|URL... | [VERSION | FILTER]
```

### Examples

```

geneos install gateway
geneos install fa2 5.5 -U
geneos install netprobe -b active_dev -U

```

### Options

```
  -b, --base string       Override the base active_prod link name (default "active_prod")
  -L, --local             Install from local files only
  -n, --nosave            Do not save a local copy of any downloads
  -H, --host string       Perform on a remote host. "all" means all hosts and locally (default "all")
  -N, --nexus             Download from nexus.itrsgroup.com. Requires auth.
  -p, --snapshots         Download from nexus snapshots (pre-releases), not releases. Requires -N
  -V, --version string    Download this version, defaults to latest. Doesn't work for EL8 archives. (default "latest")
  -u, --username string   Username for downloads, defaults to configuration value in download.username
  -P, --pwfile string     Password file to read for downloads, defaults to configuration value in download.password or otherwise prompts
  -U, --update            Update the base directory symlink
  -T, --override string   Override (set) the TYPE:VERSION for archive files with non-standard names
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

