Initialise a Geneos Demo environment, creating a new directory structure
as required.

Without any flags the command installs the components in a directory
called `geneos` under the user's home directory (unless the user's home
directory ends in `geneos` in which case it uses that directly),
downloads the latest release archives and creates a Gateway instance
using the name `Demo Gateway` (with embedded space) as required for Demo
licensing, as Netprobe and a Webserver.

If the release archive files required have already been downloaded then
use the `-A directory` flag to indicate their location. For each
component type this directory is checked for the latest release.

Otherwise, to fetch the releases from the ITRS download server
authentication will be required use the `-u email@example.com` to
specify the user account and you will be prompted for a password.

The initial configuration file for the Gateway is built from the default
templates installed and located in `.../templates` but this can be
overridden with the `-s` option. For the Gateway you can add include
files using `-i PRIORITY:PATH` flag. This can be repeated multiple
times.

Other flags inherited from the `geneos init` command can be used to
influence the installation.
