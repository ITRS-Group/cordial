The `geneos init templates` command is used to write out the default template files for a Geneos installation. Existing template files will be overwritten. Use this command when you adopt a legacy installation or upgrade the `geneos` binary and want to ensure you have the latest default templates.

The `geneos` binary contains embedded template files that are normally written out during initialization of a new installation so that they can be customised if required.

This command will overwrite any files with the same name but will not delete other template files that may already exist.

Use this command if you get missing template errors using the `geneos add` or `geneos deploy` commands.
