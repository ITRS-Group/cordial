# `gdna start`

Use `gdna start` to start a process that acquires, processes and reports data as well as being able to optionally send email reports on a schedule.

With the `--daemon`/`-D` flag the process runs in the background, writing its log to the file in the configuration, and the command only returns, printing the PID, once the process is running its schedules, or has failed to start. Logging to the console (`-l -`) is not possible in the background.

While it runs the process holds a PID file, which stops a second process starting with the same file and is how the `gdna stop`, `gdna restart`, `gdna reload` and `gdna status` commands find it. The file is set with the `--pidfile` option or `gdna.pidfile` in the configuration. If neither is set then a background process uses `gdna.pid` in the `geneos` directory in your user cache directory and a foreground process does not use one. If you run more than one process on a server then give each its own PID file, and give the same `--pidfile` option or configuration file to the commands that control it. The PID file also records how the process was started, so that `gdna restart` can start it again in the same way.

The `gdna reload` command, or sending the process the `SIGUSR1` signal, makes the process read its configuration again, if it is valid, and use the new schedules and settings for all later runs without stopping. A fetch or report that is running completes using the configuration it started with. Changes to the database, the logging and the PID file need a `gdna restart`. Reload and the `SIGUSR1` signal are not available on Windows. The `SIGTERM` signal, like `gdna stop`, and CTRL+C stop the process.

A single run with `--once` does not use a PID file and can be run while a background process is running.

## Usage

```text
gdna start
```

### Options

```text
  -D, --daemon[=true|false]           Daemonise the process
      --pidfile file                  Use the PID file, overriding `gdna.pidfile` in the configuration
  -1, --once[=true|false]             Run once and exit
  -O, --on-start[=true|false]         Run immediately on start-up, then follow schedule
  -E, --on-start-email[=true|false]   Run immediately on start-up, send email report, then follow schedule
  -r, --reports string                Run only the matching reports, for multiple reports use a
                                      comma-separated list. Report names can include shell-style wildcards.
                                      Split reports can be suffixed with ':value' to limit the report
                                      to the value given.
  -H, --hostname hostname             Connect to netprobe at hostname (default "localhost")
  -P, --port port                     Connect to netprobe on port (default 7036)
  -S, --secure[=true|false]           Use TLS connection to Netprobe
  -k, --skip-verify[=true|false]      Skip certificate verification for Netprobe connections
  -e, --entity Entity                 Send reports to Managed Entity (default "GDNA")
  -s, --sampler Sampler               Send reports to Sampler (default "GDNA")
  -R, --reset[=true|false]            Reset/Delete configured Dataviews on first run
  -f, --config FILE                   Use configuration FILE
  -l, --logfile file                  Write logs to file. Use '-' for console or /dev/null for none (default "gdna.log")
```

## SEE ALSO

* [gdna](gdna.md)	 - Process Geneos License Usage Data
