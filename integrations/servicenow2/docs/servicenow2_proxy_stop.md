# `servicenow2 proxy stop`

Stop the servicenow2 proxy that is running in the background.

The process is asked to stop, and it then stops accepting requests and
waits for any that are in progress to complete before it exits.
If it has not stopped within the timeout then this command fails, unless
--force is used, which kills it.

The process is found using its PID file.

## Usage

```text
servicenow2 proxy stop [flags]
```

### Options

```text
  -c, --conf string          override config file
      --force[=true|false]   Kill the process if it has not stopped within the timeout
  -l, --logfile file         Write logs to file. Use '-' for console or /dev/null for none (default "-")
      --pidfile file         Use the PID file, overriding `server.pidfile` in the configuration
      --timeout duration     Wait this long for the process to stop (default 30s)
```

## SEE ALSO

* [servicenow2 proxy](servicenow2_proxy.md)	 - Run a ServiceNow integration proxy
