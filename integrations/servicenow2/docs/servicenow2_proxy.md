# `servicenow2 proxy`


Run an ITRS Geneos to ServiceNow proxy.

The proxy acts as a proxy between Geneos Gateways, each running an
incident submission client, and the ServiceNow instance API. The
proxy can run on a different network endpoint, such as a DMZ, and
can also help limit the number of IP endpoints connecting to a
ServiceNow instance that may have a limit on source connections. The
proxy can also act on data fetched from ServiceNow as part of the
incident submission or update flow.

In normal operation the proxy starts and runs in the foreground,
logging actions and results to stdout/stderr. If started with the
`--daemon` flag it will background itself, writing its log to a file, and
only return once it is listening or has failed to start. It holds a PID
file while it runs, which is the one given with `--pidfile`, or set as
`server.pidfile` in the configuration, or if neither a file in the
user's cache directory when it is a background process.

A proxy running in the background is controlled by the `stop`,
`restart`, `reload` and `status` sub-commands. `reload` makes the proxy
read its configuration again, if it is valid, without interrupting any
requests that are in progress. This is also done by sending the process
the `SIGUSR1` signal. Changes to the logging settings need a restart.

The proxy reads it's configuration from a YAML file, which can be
shared with the submission client function, and uses this to look-up,
map and submit incidents.

## Usage

```text
servicenow2 proxy [flags]
```

## Commands

| Command | Description |
|-------|-------|
| [`servicenow2 proxy reload`](servicenow2_proxy_reload.md)	 | Reload the configuration of the background servicenow2 proxy |
| [`servicenow2 proxy restart`](servicenow2_proxy_restart.md)	 | Restart the background servicenow2 proxy |
| [`servicenow2 proxy status`](servicenow2_proxy_status.md)	 | Show whether the background servicenow2 proxy is running |
| [`servicenow2 proxy stop`](servicenow2_proxy_stop.md)	 | Stop the background servicenow2 proxy |

### Options

```text
  -D, --daemon[=true|false]   Daemonise the proxy process
      --pidfile file          Use the PID file, overriding `server.pidfile` in the configuration
  -l, --logfile file          Write logs to file. Use '-' for console or /dev/null for none (default "-")
  -c, --conf string           override config file
```

## SEE ALSO

* [servicenow2](servicenow2.md)	 - Geneos to ServiceNow integration
