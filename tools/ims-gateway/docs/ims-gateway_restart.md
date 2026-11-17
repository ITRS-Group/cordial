# `ims-gateway restart`

Stop the ims-gateway that is running in the background, as the stop command
does, and start it again in the background, with the same arguments as it
was started with, which are recorded in its PID file. This also uses a new
version of the program if it has been upgraded.

To pick up a changed configuration without stopping, use the reload
command instead.

## Usage

```text
ims-gateway restart [flags]
```

### Options

```text
  -c, --conf string          override config file
      --force[=true|false]   Kill the process if it has not stopped within the timeout
      --pidfile file         Use the PID file, overriding `server.pidfile` in the configuration
      --timeout duration     Wait this long for the process to stop (default 30s)
```

## SEE ALSO

* [ims-gateway](ims-gateway.md)	 - ITRS IMS Gateway
