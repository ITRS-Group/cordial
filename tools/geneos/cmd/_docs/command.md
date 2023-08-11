Show the command line and environment variables for matching instances.

If given the `--json`/`-j` flag then the output is an array of objects with the details of each instance and the command environment used.

The options `--extras`/`-x` and `--env`/`-e` can be used to add one-off extra command line parameters and environment variables to the command constructors. This is primarily to match the `start` and `restart` commands and for diagnostics and debugging
