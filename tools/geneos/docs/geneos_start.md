# `geneos start`

Start Instances

```text
geneos start [flags] [TYPE] [NAME...]
```

Start the matching instances.

The start-up command and environment can be seen using the `geneos command` command.

Any matching instances that are marked as `disabled` are not started.

With the `--log`/`-l` option the command will follow the logs of all instances started, including the STDERR logs as these are good sources of start-up issues.

### Options

```text
  -l, --log   Follow logs after starting instance
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
