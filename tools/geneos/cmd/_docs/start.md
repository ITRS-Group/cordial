The `start` command starts the matching instances unless they are already running (or marked as `disabled`). Use the `restart --all` command to restart all instances regardless of their current state.

The start-up command and environment can be seen using the `geneos command` command.

Any matching instances that are marked as `disabled` are not started. If an instance has a parameter `autostart` set to `false` then this is only started if the name is given explicitly on the command line, e.g. `geneos start gateway Example` but not `geneos start gateway` or `geneos start all`. This allows you to have instances that are not started by default but can be started when needed without having to change the configuration.

With the `--log`/`-l` option the command will follow the logs of all instances started, including the STDERR logs as these are good sources of start-up issues.

The options `--extras`/`-x` and `--env`/`-e` can be used to add one-off extra command line parameters and environment variables to the start-up of the process. This can be useful when you may need to run a Gateway with an option like `-skip-cache` after rotating key-files, e.g. `geneos start gateway Example -x -skip-cache`.
