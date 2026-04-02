# `geneos incident query`

# `geneos incident query`

Query for a list of incidents and their details. This command is used to query the `tools/ims-gateway` program for a list of incidents and their details. The command can be used to filter the list of incidents based on various criteria such as status, priority, assignee, etc. The command can also be used to display the details of a specific incident by providing the incident ID.

The command relies on a configuration file, normally locates in `${HOME}/.config/geneos/ims.yaml`, to provide the connection details for the `ims-gateway` program. If the configuration file is not found or is invalid then an error will be returned. You can specify an alternative configuration file using the `--config`/`-C` option.

```text
geneos incident query [FLAGS] [flags]
```

### Options

```text
  -i, --ims string          IMS type, e.g. "snow" or "sdp". default taken from config file
  -T, --snow-table string   ServiceNow table, defaults to incident
  -R, --snow-raw            turn ServiceNow sys_display off, i.e. return raw values instead of display values
  -Q, --query string        query to use for the specified IMS type, e.g. a ServiceNow encoded query or a ServiceDesk Plus JSON query. default taken from config file
  -f, --format csv          output format: csv or json (default "csv")
```

## SEE ALSO

* [geneos incident](geneos_incident.md)	 - Commands for working with incidents
