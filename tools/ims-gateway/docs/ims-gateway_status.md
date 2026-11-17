# `ims-gateway status`

Show whether the ims-gateway is running in the background, using its PID file.
The exit code is zero only if it is running.

## Usage

```text
ims-gateway status [flags]
```

### Options

```text
  -c, --conf string    override config file
      --pidfile file   Use the PID file, overriding `server.pidfile` in the configuration
```

## SEE ALSO

* [ims-gateway](ims-gateway.md)	 - ITRS IMS Gateway
