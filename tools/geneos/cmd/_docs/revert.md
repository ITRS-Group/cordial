The command will revert the `.rc.orig` suffixed configuration file for all matching instances.

For any instance that is `protected` this will fail and an error reported.

The original file is never updated and any changes made since the original migration will be lost. The new configuration file will be deleted.

If there is already a configuration file with a `.rc` suffix then the command will remove any `.rc.orig` and new configuration files while leaving the existing file unchanged.

If called with the `--executables`/`-X` option then instead of instance configurations the command will remove any symbolic links from legacy `ctl` command in `${GENEOS_HOME}/bin` that point to the command.
