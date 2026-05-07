# `geneos incident`

The `incident` subsystem allows you to manage the creation, update and resolution of incidents via the `tools/ims-gateway` program. You can also query for a list of incidents and their details.

Currently this incident subsystem, via the `ims-gateway` program, supports ServiceNow and ServiceDesk Plus.

## Usage

```text
geneos incident
```

## Commands

| Command | Description |
|-------|-------|
| [`geneos incident query`](geneos_incident_query.md)	 | Query IMS |
| [`geneos incident raise`](geneos_incident_raise.md)	 | Create or update an incident |

### Options

```text
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
