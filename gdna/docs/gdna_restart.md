# `gdna restart`

Stop the gdna that is running in the background, as the stop command
does, and start it again in the background, with the same arguments as it
was started with, which are recorded in its PID file. This also uses a new
version of the program if it has been upgraded.

To pick up a changed configuration without stopping, use the reload
command instead.

## Usage

```text
gdna restart
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
