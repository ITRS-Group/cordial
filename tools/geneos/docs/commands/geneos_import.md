# `geneos import`

Import files to an instance or a common directory

```text
geneos import [flags] [TYPE] [NAME...] [DEST=]SOURCE...
```

## Details

Import each `SOURCE` to instance directories. With the `--common`/`-c`
option the imports are to a TYPE component sub-directory `TYPE_`
suffixed with the value to the `--common`/`-c` option. See examples
below.

The `SOURCE` can be the path to a local file, a URL or '-' for `STDIN`.
`SOURCE` may not be a directory.

If `SOURCE` is a file in the current directory then it must be prefixed
with `"./"` to avoid being seen as an instance NAME to search for. Any
file path with a directory separator already present does not need this
precaution. The program will read from `STDIN` if the `SOURCE` '-' is
given but this can only be used once and a destination DEST must be
defined.

If `DEST` is given with a `SOURCE` then it must either be a plain file
name or a descending relative path. An absolute or ascending path is an
error.

Without an explicit `DEST` for the destination file only the base name
of the `SOURCE` is used. If `SOURCE` is a URL then the file name for the
resource from the remote web server is preferred over the last part of
the URL.

If the `--common`/`-c` option is used then a TYPE must also be
specified. Each component of TYPE has a base directory. That directory
may contain, in addition to instances of that TYPE, a number of other
directories that can be used for shared resources. These may be scripts,
include files and so on. Using a TYPE `gateway` as an example and using
a `--common config` option the destination for `SOURCE` would be
`gateway/gateway_config`

Future releases may add support for directories and.or unarchiving of
`tar.gz`/`zip` and other file archives.

### Options

```text
  -c, --common SUFFIX   Import files to a component directory named TYPE_SUFFIX
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## Examples

```bash
# import a gateway setup file from a web server
geneos import gateway example1 https://example.com/myfiles/gateway.setup.xml

# import the "license.txt" file to the licd instance example2 but
# with a filename of geneos.lic
geneos import licd example2 geneos.lic=license.txt

# import the "myscript.sh" file into the scripts directory under the
# netprobe example3's working directory
#
# Note: the file will not be made executable
geneos import netprobe example3 scripts/=myscript.sh

# import the file "netprobe.setup.xml" from the current directory to
# the SAN localhost
# 
# Note the leading "./" to disambiguate the file name from an instance
# to match
geneos import san localhost ./netprobe.setup.xml

# import "common_include" into the gateway_shared directory under the gateway are of the installation directory
geneos import gateway -c shared common_include.xml

```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
