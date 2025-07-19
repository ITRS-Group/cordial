# `geneos load` / `geneos restore`

Load one or more instances from an archive created by `geneos save` including shared directories when using `--shared-s`.

The command accepts a combination of filenames and instance name patterns, with optional renaming prefixes, and distinguishes them by validating the arguments. Any arguments that are not valid instance names (or wildcard or rename patterns) are treated as archive files. In case your archive file matches a valid instance name you should either use an absolute path to the file or a `./` prefix.

