## geneos package install

Install Geneos releases

### Synopsis


Installs Geneos software packages in the Geneos directory structure under
directory `packages`. The Geneos software packages will be sourced from
the [ITRS Download portal](https://resources.itrsgroup.com/downloads) or,
if specified as FILE or URL, a filename formatted as
`geneos-TYPE-VERSION*.tar.gz`.

Installation will ...
- Select the latest available version, or the version specified with option
  `-V <version>`.
- Store the packages downloaded from the ITRS Download portal into 
  `packages/downloads`, unless the `-n` option is selected.
  In case a FILE or URL is specified, the FILE or URL will be used as the
  packages source and nothing will be written to `packages/downloads`.
- Place binaries for TYPE into `packages/<TYPE>/<version>`, where
  <version> is the version number of the package and can be overridden by
  using option `-T`.
- in case no symlink pointing to the default version exists, one will be
  created as `active_prod` or using the name provided with option 
  `-b <symlink_name>`.
  **Note**: Option `-b <symlink_name>` may be used in conjunction with `-U`.
- If option `-U` is used, the symlink will be updated.
  If this is used, instances of the binary will be stopped and restarted
  after the link has been updated.

  The `geneos install` command works for the following component types:
  `licd` (license daemon), `gateway`, `netprobe`, `webserver` (webserver for 
  web dashboards), `fa2` (fix analyser netprobe), `fileagent` (file agent for 
  fix analyser).


```
geneos package install [flags] [TYPE] [FILE|URL...]
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

* [geneos package](geneos_package.md)	 - A brief description of your command

