# `geneos start`

Start the matching instances.

The start-up command and environment can be seen using the `geneos command` command.

Any matching instances that are marked as `disabled` are not started.

With the `--log`/`-l` option the command will follow the logs of all instances started, including the STDERR logs as these are good sources of start-up issues.

The options `--extras`/`-x` and `--env`/`-e` can be used to add one-off extra command line parameters and environment variables to the start-up of the process. This can be useful when you may need to run a Gateway with an option like `-skip-cache` after rotating key-files, e.g. `geneos start gateway Example -x -skip-cache`.

```text
geneos start [flags] [TYPE] [NAME...]
```

### Options

```text
  -x, --extras string    Extra args passed to process, split on spaces and quoting ignored
  -e, --env NAME=VALUE   Extra environment variable (Repeat as required)
  -l, --log              Follow logs after starting instance
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
