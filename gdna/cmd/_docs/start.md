Use `gdna start` to start a process that acquires, processes and reports data as well as being able to optionally send email reports on a schedule.

With the `--daemon`/`-D` flag the process runs in the background, writing its log to the file in the configuration, and the command only returns, printing the PID, once the process is running its schedules, or has failed to start. Logging to the console (`-l -`) is not possible in the background.

While it runs the process holds a PID file, which stops a second process starting with the same file and is how the `gdna stop`, `gdna restart`, `gdna reload` and `gdna status` commands find it. The file is set with the `--pidfile` option or `gdna.pidfile` in the configuration. If neither is set then a background process uses `gdna.pid` in the `geneos` directory in your user cache directory and a foreground process does not use one. If you run more than one process on a server then give each its own PID file, and give the same `--pidfile` option or configuration file to the commands that control it. The PID file also records how the process was started, so that `gdna restart` can start it again in the same way.

The `gdna reload` command, or sending the process the `SIGUSR1` signal, makes the process read its configuration again, if it is valid, and use the new schedules and settings for all later runs without stopping. A fetch or report that is running completes using the configuration it started with. Changes to the database, the logging and the PID file need a `gdna restart`. Reload and the `SIGUSR1` signal are not available on Windows. The `SIGTERM` signal, like `gdna stop`, and CTRL+C stop the process.

A single run with `--once` does not use a PID file and can be run while a background process is running.

