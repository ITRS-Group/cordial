# `gdna reload`

Make the gdna that is running in the background read its configuration
again, without stopping it.

The configuration is first checked by this command, using the file given
on the command line or, if there is not one, the file that the process was
started with, and the process is only told to reload if it is usable.
The process checks it again and, if it is not usable, continues with the
configuration it has.

A fetch or report that is running when the configuration is reloaded
completes using the configuration it started with, and the schedules and all
later runs use the new one. Changes to the database, the logging and the PID
file need a restart.

The log of the process shows the result.

Reload is not available on Windows, use restart.

## Usage

```text
gdna reload
```

### Options

```text
  -f, --config FILE    Use configuration FILE
  -l, --logfile file   Write logs to file. Use '-' for console or /dev/null for none (default "gdna.log")
      --pidfile file   Use the PID file, overriding `gdna.pidfile` in the configuration
```

## SEE ALSO

* [gdna](gdna.md)	 - Process Geneos License Usage Data
