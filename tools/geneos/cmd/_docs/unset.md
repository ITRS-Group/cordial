Unset (remove) configuration parameters from matching instances. This
command is `unset` rather than `remove` as that is reserved as an
alias for the `delete` command.

Unlike the `geneos set` command, where parameters are in the form of
KEY=VALUE, there is no way to distinguish a KEY to remove and a
possible instance name, so you must use one or more `--key`/`-k`
options to unset each simple parameter.

WARNING: Be careful removing keys that are necessary for instances to
be manageable. Some keys, if removed, will require manual
intervention to remove or fox the old configuration and recreate the
instance.

You can also unset values for structured parameters. For
`--include`/`-i` options the parameter key is the `PRIORITY` of the
include file set while for the other options it is the `NAME`. Note
that for structured parameters the `NAME` is case-sensitive.
