# `ims-gateway stop`

Stop the ims-gateway that is running in the background.

The process is asked to stop, and it then stops accepting requests and
waits for any that are in progress to complete before it exits.
If it has not stopped within the timeout then this command fails, unless
--force is used, which kills it.

The process is found using its PID file.

## Usage

```text
ims-gateway stop [flags]
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
