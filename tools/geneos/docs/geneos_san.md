# `geneos san`

# `geneos` Self-Announcing Netprobe Comopnents

A Self-Announcing Netprobe (SAN) uses the standard Netprobe installation
package but announces itself to the Gateway and carries much of it's
own configuration in a local setup XML file. Multiple SANs can connect
to a Gateway as long as each has it's own, unique name.

* San general

* San templates

* San variables for templates

  Like for Gateways, SANs get a default configuration file when they are
  created. By default this is from the template(s) in `san/templates`.
  Unlike for the Gateway these configuration files are rebuilt by the
  `rebuild` command by default. This allows the administrator to
  maintain SANs using only command line tools and avoid having to edit
  XML directly. Setting `configrebuild` to `never` in the instance
  configuration prevents this rebuild. To aid this, SANs support the
  following special parameters:

  * Attributes

  Attributes can be added via `set`, `add` or `init` using the `-a` flag
  in the form NAME=VALUE and also removed using `unset` in the same way
  but just with a NAME

  * Gateways

  As for Attributes, the `-g` flag can specify Gateways to connect to in
  the form HOSTNAME:PORT

  * Types

  Types can be specified using `-t`

  * Variables

  Variables can be set using `-v` but there is only support for a
  limited number of types, specifically those that have values that can
  be give in plain string format.

* Selecting the underlying Netprobe type (For Fix Analyser 2 below) A
  San instance will normally be built to use the general purpose
  Netprobe package. To use an alternative package, such as the Fix
  Analyser 2 Netprobe, add the instance with the special format name
  `fa2:example[@REMOTE]` - this configures the instance to use the `fa2`
  as the underlying package. Any future special purpose Netprobes can
  also be supported in this way.


* Secure Connections

If a "certificate", a "chainfile" or the "secure" parameters exist then
Gateway connections are done using TLS.
```text
geneos san
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
