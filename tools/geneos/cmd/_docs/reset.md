The `reset` command removes files and directories in the matching instance working directory to a near-default state. The instances are stopped, the directories cleaned and the instance restarted. Because the command can have side-effects, to match all instances you must provide the `all` keyword as `geneos reset` will not match any instances by default.

For instances that are "protected" you must use the `--force`/`-F` option to reset the instance even if it is running.

The files and directories that are removed are based on the component type and the global settings using the `clean` and `purge` pattern lists.
