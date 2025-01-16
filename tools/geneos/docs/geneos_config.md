# `geneos config`

# `geneos config`

The config sub-system allow you to control the environment of the `geneos` program itself.

## General Configuration

* `/etc/geneos/geneos.json` - Global options
* `${HOME}/.config/geneos/geneos.json` - User options
* Environment variables ITRS_`option` - where `.` is replaced by `_`, e.g. `ITRS_DOWNLOAD_USERNAME`

General options are loaded from the global file first, then the user file and any environment variables override both files. The currently supported options are:

* `geneos`

The home directory for all other commands. See [Directory Layout](#directory-layout) below. If set, the environment variable ITRS_HOME overrides any settings in the files. This is to maintain backward compatibility with older tools. The default, if not set anywhere else, is the home directory of the user running the command or, if running as root, the home directory of the `geneos` or `itrs` users (in that order). (To be fully implemented) This value is also set by the environment variables `ITRS_HOME` or `ITRS_GENEOS`

* `download.url`

The base URL for downloads for automating installations. Not yet used. If files are locally downloaded then this can either be a `file://` style URL or a directory path.

* `download.username` `download.password`

  These specify the username and password to use when downloading packages. They can also be set as the environment variables, but the environment variables are not subject to expansion and so cannot contain Geneos encoded passwords (see below):

    * `ITRS_DOWNLOAD_USERNAME`
    * `ITRS_DOWNLOAD_PASSWORD`

* `snapshot.username` `snapshot.password`

  Similarly to the above, these specify the username and password to use when taking dataview snapshots. They can also be set as the environment variables, with the same restrictions as above:

    * `ITRS_SNAPSHOT_USERNAME`
    * `ITRS_SNAPSHOT_PASSWORD`

* `GatewayPortRange` & `NetprobePortRange` & `LicdPortRange`


## Commands

| Command | Description |
|-------|-------|
| [`geneos config export`](geneos_config_export.md)	 | Export Instances |
| [`geneos config import`](geneos_config_import.md)	 | import Instances |
| [`geneos config set`](geneos_config_set.md)	 | Set program configuration |
| [`geneos config show`](geneos_config_show.md)	 | Show program configuration |
| [`geneos config unset`](geneos_config_unset.md)	 | Unset a program parameter |

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
