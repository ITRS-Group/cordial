# `gdna status`

Show whether the gdna is running in the background, using its PID file.
The exit code is zero only if it is running.

## Usage

```text
gdna status
```

### Options

```text
  -f, --config FILE    Use configuration FILE
  -l, --logfile file   Write logs to file. Use '-' for console or /dev/null for none (default "docs.log")
      --pidfile file   Use the PID file, overriding `gdna.pidfile` in the configuration
```

## SEE ALSO

* [gdna](gdna.md)	 - Process Geneos License Usage Data
