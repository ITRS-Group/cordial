# `geneos` Components

## Component Types

The following component types (and their aliases) are supported:

* **`gateway`** - or `gateways`

  A Geneos Gateway

* **`netprobe`** - or `netprobes`, `probe` or `probes`

* **`san`** - or `sans`

* **`floating`** - or `float`

* **`ca3`** - `collection-agent`, `collector` or `ca3s`

* **`licd`** - or `licds`

* **`webserver`** - or `webservers`, `webdashboard`. `dashboards`

* **`fa2`** - or `fixanalyser`, `fix-analyser`

* **`fileagent`** - or `fileagents`

* `any` (which is the default)

The first name, in bold, is also the directory name used for each type.
These names are also reserved words and you cannot configure (or expect
to consistently manage) components with those names. This means that you
cannot have a gateway called `gateway` or a probe called `probe`. If you
do already have instances with these names then you will have to be
careful migrating. See more below.

Each component type is described below along with specific component options.

**Note** This section is not yet complete, apologies.

### Type `gateway`

* Gateway general

* Gateway templates

  When creating a new Gateway instance a default `gateway.setup.xml`
  file is created from the template(s) installed in the
  `gateway/templates` directory. By default this file is only created
  once but can be re-created using the `rebuild` command with the `-F`
  option if required. In turn this can also be protected against by
  setting the Gateway configuration setting `configrebuild` to `never`.

* Gateway variables for templates

  Gateways support the setting of Include files for use in templated
  configurations. These are set similarly to the `-e` parameters:

  ```bash
  geneos gateway set example2 -i  100:/path/to/include
  ```

  The setting value is `priority:path` and path can be a relative or
  absolute path or a URL. In the case of a URL the source is NOT
  downloaded but instead the URL is written as-is in the template
  output.

### Type `netprobe`

* Netprobe general

### Type `licd`

* Licd general

### Type `webserver`

* Webserver general

* Java considerations

* Configuration templates - TBD

### Type `san`

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

### Type `fa2`

* Fix Analyser 2 general

### Type `fileagent`

* File Agent general