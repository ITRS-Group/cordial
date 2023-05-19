Add a new instance of a component TYPE with the name NAME.

The meaning of the options vary by component TYPE and are stored in a
configuration file in the instance directory.

The default configuration file format and extension is `json`. There
will be support for `yaml` in future releases.
	
The instance will be started after completion if given the
`--start`/`-S` or `--log`/`-l` options. The latter will also follow the
log file until interrupted.

Geneos components all use TCP ports either for inbound connections or,
in the case of SANs, to identify themselves to the Gateway. The program
will choose the next available port from the list in the for each
component called `TYPEportrange` (e.g. `gatewayportrange`) in the
program configurations. Availability is only determined by searching all
other instances (of any TYPE) on the same host. This behaviour can be
overridden with the `--port`/`-p` option.

When an instance is started it is given an environment made up of the
variables in it's configuration file and some necessary defaults, such
as `LD_LIBRARY_PATH`.  Additional variables can be set with the
`--env`/`-e` option, which can be repeated as many times as required.

The underlying package used by each instance is referenced by a
`basename` which defaults to `active_prod`. You may want to run multiple
components of the same type but different releases. You can do this by
configuring additional base names with `geneos package update` and by
setting the base name with the `--base``-b` option.

Gateways, SANs and Floating probes are given a configuration file based
on the templates configured for the different components. The default
template can be overridden with the `--template`/`-T` option specifying
the source to use. The source can be a local file, a URL or `STDIN`.

Any additional command line arguments are used to set configuration
values. Any arguments not in the form NAME=VALUE are ignored. Note that
NAME must be a plain word and must not contain dots (`.`) or double
colons (`::`) as these are used as internal delimiters. No component
uses hierarchical configuration names except those that can be set by
the options above.

You can select the type of SAN or Floating Netprobe using the special
syntax for the `NAME` in the form `TYPE:NAME`. The only supported `TYPE`
at the moment, in addition to the default `netprobe`, is `fa2` allowing
you to deploy Fix Analyser 2 based SAN and Floating probes.
