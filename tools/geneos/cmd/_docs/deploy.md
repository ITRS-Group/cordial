Deploy a new instance of a component.

The difference between `deploy` and either `add` or `init` is that
deploy will check and create the Geneos directory hierarchy if needed,
download and/or install packages for the component type and add the
instance, optionally starting it.

This allows you to create an instance without having to worry about
initialising the set-up etc.

There are many options and which you use depends on any existing Geneos
installation, whether you have Internet access and which component you
are deploying.

The stages that deploy goes through will help you choose the options you
need:

1. For local deployments, if there is no `GENEOS_HOME` (either in the
   user configuration file or as an environment variable) and a
   directory is given with `--geneos`/`-D` then a new Geneos
   installation hierarchy is created and your configuration file is
   created or updates with the new home directory. If the
   `--geneos`/`-D` option is given it will override any other setting.

   If the destination for the deployment is a configured remote host
   then the GENEOS_HONE path configured for that host is always used and
   the `--geneos`/`-D` option will result in an error if the path is
   different to the one configured for the remote.

2. If an existing release is installed for the component `TYPE` and the
   base link `--base`/`-b` (default `active_prod`) is present then this
   is used otherwise `deploy` will install the release selected with the
   `--version`/`-V` option (default `latest`) either from the official
   download site or from a local archive. If `--archive`/`-A` is a
   directory then it is searched for a suitable release archive using
   the standard naming convention for downloads. If you need to install
   from a specific file that does not conform to the normal naming
   conventions then you can override the TYPE and VERSION with the
   `--override`/`-o` option.

3. An instance is added with the various options available, just like
   the `add` command, with the options selected and additional
   parameters given as `NAME=VALUE` pairs on the command line.

4. If the `--start`/`-S` or `--log`/`-l` options are given then the new
   instance is started.




Add a new instance of a component `TYPE` with the name `NAME`.

The applicability of the options vary by component `TYPE` and are stored
in a configuration file in the instance directory.

The default configuration file format and extension is `json`. There
will be support for `yaml` in future releases.
	
The instance will be started after being added if the `--start`/`-S` or
`--log`/`-l` option is used. The latter will also follow the log file
until interrupted.

Geneos components all use TCP ports either for inbound connections or,
in the case of SANs, to identify themselves to the Gateway. The program
will choose the next available port from the list in the for each
component called `TYPEportrange` (e.g. `gatewayportrange`) in the
program configurations. Availability is determined by searching the
configurations od all other instances (of any `TYPE`) on the same host.
This behaviour can be overridden with the `--port`/`-p` option.

When an instance is started it has an environment made up of the
variables in it's configuration file and some necessary defaults, such
as `LD_LIBRARY_PATH`. Additional variables can be set with the
`--env`/`-e` option, which can be repeated as many times as required.

The underlying package used by each instance is referenced by a
`basename` parameter which defaults to `active_prod`. You can run
multiple components of the same type but different releases. You can do
this by configuring additional base names in advance with `geneos
package update` and then by setting the base name with the `--base`/`-b`
option.

Gateways, SANs and Floating probes are given a configuration file based
on the templates configured for the different components. The default
template can be overridden with the `--template`/`-T` option specifying
the source to use. The source can be a local file, a URL or `STDIN`.

Any additional command line arguments are used to set configuration
values. Any arguments not in the form `NAME=VALUE` are ignored. Note
that `NAME` must be a plain word and must not contain dots (`.`) or
double colons (`::`) as these are used as internal delimiters. No
component uses hierarchical configuration names except those that can be
set by the options above.

You can select the distribution of SAN or Floating Netprobe using the
special syntax for the `NAME` in the form `TYPE:NAME`. The only
supported `TYPE` at the moment, in addition to the default `netprobe`,
is `fa2` allowing you to deploy Fix Analyser 2 based SAN and Floating
probes.
