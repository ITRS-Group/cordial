## servicenow incident

Raise or update a ServiceNow incident

### Synopsis


Raise or update a ServiceNow incident from ITRS Geneos.

This command is the client-side of the ITRS Geneos to ServiceNow
incident integration. The program takes command line flags, arguments
and environment variables to create a submission to the router
instance which is responsible for sending the request to the
ServiceNow API.




```
servicenow incident [flags]
```

### Options

```
  -s, --short string      short description
  -t, --text string       Textual note. Long desceription for new incidents, Work Note for updates.
      --rawtext string    Raw textual note, not unquoted. Long desceription for new incidents, Work Note for updates.
  -i, --id string         Correlation ID. The value is hashed to a 20 byte hex string.
      --rawid string      Raw Correlation ID. The value is passed as is and must be a valid string.
  -f, --search string     sysID search: '[TABLE:]FIELD=VALUE', TABLE defaults to 'cmdb_ci'. REQUIRED
  -S, --severity string   Geneos severity. Maps depending on configuration settings. (default "3")
  -U, --updateonly        If set no incident creation will be done
```

### Options inherited from parent commands

```
  -c, --conf string   override config file
```

### SEE ALSO

* [servicenow](servicenow.md)	 - Geneos to ServiceNow integration

