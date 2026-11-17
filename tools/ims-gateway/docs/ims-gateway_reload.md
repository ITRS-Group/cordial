# `ims-gateway reload`

Make the ims-gateway that is running in the background read its configuration
again, without stopping it.

The configuration is first checked by this command, using the file given
on the command line or, if there is not one, the file that the process was
started with, and the process is only told to reload if it is usable.
The process checks it again and, if it is not usable, continues with the
configuration it has.

Requests that are in progress complete using the configuration they
started with and all others use the new one.

The log of the process shows the result.

Reload is not available on Windows, use restart.

## Usage

```text
ims-gateway reload [flags]
```

### Options

```text
  -c, --conf string    override config file
      --pidfile file   Use the PID file, overriding `server.pidfile` in the configuration
```

## SEE ALSO

* [ims-gateway](ims-gateway.md)	 - ITRS IMS Gateway
