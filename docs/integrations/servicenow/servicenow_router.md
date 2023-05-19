# `servicenow router`

Run a ServiceNow integration router

```text
servicenow router [flags]
```

## Details


Run an ITRS Geneos to ServiceNow router.

The router acts as a proxy between Geneos Gateways, each running an
incident submission client, and the ServiceNow instance API. The
router can run on a different network endpoint, such as a DMZ, and
can also help limit the number of IP endpoints connecting to a
ServiceNow instance that may have a limit on source connections. The
router can also act on data fetched from ServiceNow as part of the
incident submission or update flow.

In normal operation the router starts and runs in the foreground,
logging actions and results to stdout/stderr. If started with eh
`--daemon` flag it will background itself and no logging will be
available. (Logging to an external file will be added in a future
release)

The router reads it's configuration from a YAML file, which can be
shared with the submission client function, and uses this to look-up,
map and submit incidents.


### Options

```text
  -D, --daemon   Daemonise the router process
```

### Options inherited from parent commands

```text
  -c, --conf string   override config file
```

## SEE ALSO

* [servicenow](servicenow.md)	 - Geneos to ServiceNow integration
