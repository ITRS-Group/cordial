# `servicenow2 proxy status`

Show whether the servicenow2 proxy is running in the background, using its PID file.
The exit code is zero only if it is running.

## Usage

```text
servicenow2 proxy status [flags]
```

### Options

```text
  -c, --conf string    override config file
  -l, --logfile file   Write logs to file. Use '-' for console or /dev/null for none (default "-")
      --pidfile file   Use the PID file, overriding `server.pidfile` in the configuration
```

## SEE ALSO

* [servicenow2 proxy](servicenow2_proxy.md)	 - Run a ServiceNow integration proxy
