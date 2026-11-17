# `servicenow2 client`


	Create or update a ServiceNow incident from ITRS Geneos.
	
	This command is the client-side of the ITRS Geneos to ServiceNow
	incident integration. The program takes command line flags, arguments
	and environment variables to create a submission to the proxy
	which is responsible for sending the request to the
	ServiceNow API.

	The exit code is zero only if a proxy accepted the request and
	reported that the incident was created, updated or deliberately
	ignored. In all other cases an error is written to standard error
	and the exit code is non-zero.
	

## Usage

```text
servicenow2 client [FLAGS] [field=value ...] [flags]
```

### Options

```text
  -p, --profile string       profile to use for field creation
  -t, --table string         servicenow table, defaults typically to incident
  -q, --quiet[=true|false]   quiet mode. supress all non-error messages
  -c, --conf string          override config file
```

## SEE ALSO

* [servicenow2](servicenow2.md)	 - Geneos to ServiceNow integration
