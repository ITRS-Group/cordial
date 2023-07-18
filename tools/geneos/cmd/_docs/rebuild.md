All matching instances whose TYPE supported templates for configuration file will have them rebuilt depending on the `config::rebuild` setting for each instance.

The values for the `config::rebuild` option are: `never`, `initial` and `always`. The default value depends on the TYPE; For Gateways it is `initial` and for SANs and Floating Netprobes it is `always`.

You can force a rebuild for an instance that has the `config::rebuild` set to `initial` by using the `--force`/`-F` option. Instances with a `never` setting are never rebuilt.

To change this use something like `geneos set gateway MyGateway config::rebuild=always`

Instances will not normally update their settings when the configuration file changes, although there are options for both Gateways and Netprobes to do this, so you can trigger a configuration reload with the `--reload`/`-r` option. This will send the appropriate signal to matching instances regardless of the underlying configuration being updated or not.

The templates used for each `TYPE` are stored in the `templates/` directory under each `TYPE` directory, e.g `[GENEOS]/gateway/templates`. If you do not have any templates because you are adopting an existing installation then the program will use internal defaults. Use the `geneos init template` command to write out the current templates based on the built-in ones. Only files with the extension `.gotmpl` are treated as templates and directories (and their contents) are ignored.

The rules for how the files are parsed and made available are those for the Go [template.ParseGlob](https://pkg.go.dev/text/template#ParseGlob) function.
