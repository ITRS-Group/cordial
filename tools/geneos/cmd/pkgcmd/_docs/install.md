Installs Geneos software packages in the Geneos directory structure
under directory `packages`. The Geneos software packages will be sourced
from the [ITRS Download
portal](https://resources.itrsgroup.com/downloads) or, if specified as
FILE or URL, a filename formatted as `geneos-TYPE-VERSION*.tar.gz`.

Installation will ...
- Select the latest available version, or the version specified with
  option `-V <version>`.
- Store the packages downloaded from the ITRS Download portal into
  `packages/downloads`, unless the `-n` option is selected. In case a
  FILE or URL is specified, the FILE or URL will be used as the packages
  source and nothing will be written to `packages/downloads`.
- Place binaries for TYPE into `packages/<TYPE>/<version>`, where
  <version> is the version number of the package and can be overridden
  by using option `-T`.
- in case no symlink pointing to the default version exists, one will be
  created as `active_prod` or using the name provided with option `-b
  <symlink_name>`. **Note**: Option `-b <symlink_name>` may be used in
  conjunction with `-U`.
- If option `-U` is used, the symlink will be updated. If this is used,
  instances of the binary will be stopped and restarted after the link
  has been updated.

  The `geneos install` command works for the following component types:
  `licd` (license daemon), `gateway`, `netprobe`, `webserver` (webserver
  for web dashboards), `fa2` (fix analyser netprobe), `fileagent` (file
  agent for fix analyser).
