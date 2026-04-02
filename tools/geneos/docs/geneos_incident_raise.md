# `geneos incident raise`

# `geneos incident raise`

Raise or update an incident.

Used with the environment variables Geneos sets as part of an Alert or an Action, or defined on the command line as NAME=VALUE parameters, these key/value pairs are processed using the settings in the `${HOME}/.config/geneos/ims.yaml` file to determine the content of the incident to be raised or updated.

```text
geneos incident raise [FLAGS] [field=value ...] [flags]
```

### Options

```text
  -c, --config string         config file to use
  -i, --ims string            IMS type, e.g. snow or sdp. default taken from config file
  -p, --profile string        profile to use for field creation
  -t, --snow-table incident   ServiceNow table, typically incident
```

## SEE ALSO

* [geneos incident](geneos_incident.md)	 - Commands for working with incidents
