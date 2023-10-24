# `geneos init template`

The `geneos` commands contains embedded template files that are normally written out during initialization of a new installation so that they can be customised if required. In the case of adopting a legacy installation or upgrading the program you should run this command to write-out the current default templates.

This command will overwrite any files with the same name but will not delete other template files that may already exist.

Use this command if you get missing template errors using the `add` command.

```text
geneos init template [flags]
```

## SEE ALSO

* [geneos init](geneos_init.md)	 - Initialise The Installation
