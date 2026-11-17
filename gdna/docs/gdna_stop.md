# `gdna stop`

Stop the gdna that is running in the background.

The process is asked to stop and it then cancels any fetch or report that is
running before it exits.
If it has not stopped within the timeout then this command fails, unless
--force is used, which kills it.

The process is found using its PID file.

## Usage

```text
gdna stop
```

### Options

```text
  -f, --config FILE          Use configuration FILE
      --force[=true|false]   Kill the process if it has not stopped within the timeout
  -l, --logfile file         Write logs to file. Use '-' for console or /dev/null for none (default "docs.log")
      --pidfile file         Use the PID file, overriding `gdna.pidfile` in the configuration
      --timeout duration     Wait this long for the process to stop (default 30s)
```

## SEE ALSO

* [gdna](gdna.md)	 - Process Geneos License Usage Data
