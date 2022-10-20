# ITRS Geneos to Pagerduty Event Integration

This program sends ITRS Geneos Alerts as Pagerduty Events which, in turn, raise or update Incidents.

## Getting Started

You can either download a binary or build from source. The default configuration transforms Geneos Alert or Rule & Action values into sensible mappings to a Pagerduty event. You will need to supply an authentication token and an integration key.

The integration consists of one executable, by default called `pagerduty`, and a configuration file with the same name and a `.yaml` extension. If you rename the executable to suit your environment then note that the configuration files (mentioned below) also change to match. e.g. if you rename the executable `itrs-pd` then the configuration files will be `itrs-pd.yaml` etc.

Unless you need to change any of the default mappings, you can start with a simple configuration file like this:

```yaml
pagerduty:
  authtoken: [SEE BELOW]
```

Save this as `.config/geneos/pagerduty.yaml` relative to the home directory of the user running the Geneos Gateway.

In your Geneos Gateway create an Effect like this, changing the path to the binary as required:

```xml
<effect name="Pagerduty Event">
    <script>
        <exeFile>/opt/itrs/gateway/gateway_shared/pagerduty</exeFile>
        <arguments>
            <data/>
        </arguments>
        <runLocation>gateway</runLocation>
    </script>
</effect>
```

If you want to run this as an Action from a Rule then create a similar Action instead (you can replace the XML tag `effect` with `action` in the above).

Finally, using the default configuration, you will need to set a `PAGERDUTY_INTEGRATION_KEY` Geneos Attribute on the Managed Entity or Managed Entity Group you will be sending events for. See below for more details on how to obtain this for each Pagerduty Service.

## How It Works

There are very few hardwired rules in the integration and most behaviours can be changed in the configuration.

Currently there is no support for suppressed (informational) alerts. These may be added in a future release.

There is also not support for Change Events at this time. These may also be added in a future release.

### Built-in Logic

Pagerduty expects an event to either be a `trigger`, `acknowledge` or a `resolve`. The integrations built-in logic may be easiest to explain by showing you the code:

```go
switch {
    case eventType == Resolve, severity == "ok", alertType == "clear":
        action = "resolve"
        severity = "info"
    case eventType == Assign, alertType == "suspend":
        action = "acknowledge"
        severity = "info"
    default:
        action = "trigger"
}
```

As you can see above the rules that decide when to send different events are simple:

* If the sub-command (named the `eventType` above) is `Resolve` then do just that
* If the Geneos severity, via the configuration file, maps to the value "ok" or the Alert type (`_ALERT_TYPE`) is a "clear" then this is a `resolve`
* If the sub-command is `Assign` or the Alert type is a "suspend" then send an `acknowledge`
* Otherwise this is a `trigger`

For `resolve` and `acknowledge` the severity is set to `info` regardless of other settings.

The one choice that may need clarification is the selection of `acknowledge` events; From the Geneos side an Alert is suspended if the data item (or parent data item) is Snoozed or Assigned or if it becomes inactive (e.g. via an Active Time). These Geneos state changes roughly meet the criteria on the Pagerduty side for [Incident Statuses](https://support.pagerduty.com/docs/incidents) - i.e. once a Geneos data item is snoozed, assigned or inactive then it is either being worked on or not considered immediately significant. If you are using Alerts, as opposed to Actions or Commands, then these settings are reversed automatically when the item is un-snoozed.

Automated assignment/unassignment events depends on the configuration in your Gateway under Authentication -> Advanced -> User Assignment.

## Configuration Sources

The integration takes it settings from the following, in order of priority from highest to lowest:

1. Command line flags
2. Configuration files (in the order below, first found 'wins' - they are not merged)

      * `./pagerduty.yaml`
      * `${HOME}/.config/geneos/pagerduty.yaml`
      * `/etc/geneos/pagerduty.yaml`

3. External Defaults File (as above but named `pagerduty.defaults.yaml` etc.)
4. Internal Defaults

Geneos passes alert information to external programs and scripts using environment variables. These are used by the configuration options to build a Pagerduty Event in [PD-CEF](https://support.pagerduty.com/docs/pd-cef) format.

See below for the meaning of the items in the configuration.

### Configuration Logic

When run as either an Action from a Rule or an Effect from an Alert, Geneos sets a number of environment variables which can be used in the configuration to build custom Pagerduty values. The defaults given below (prefixed with an `_` underscore) are some of these. See the Geneos documentation for more details.

The default configuration sets the following:

* `authtoken` - Required

  This is the API Key that grant Geneos access to Pagerduty. You will need to create one under the API Access Keys page of your Pagerduty instance - e.g. `https://XXX.pagerduty.com/api_keys`. To protect the key from casual viewing use the `${enc:}` format supported by `cordial` via the `geneos aes` commands. For example:

  ```bash
  # unless already done, create an aes keyfile (this is save in ${HOME}/.config/geneos/keyfile.aes by default)
  $ geneos aes new -D
  # now encode the key in "expandable" format, copy the entire output line into the config
  $ geneos aes encode -e -p 'apikeyhere'
  ${enc:~/.config/geneos/keyfile.aes:+encs+6841BD18C7FB9082742B8208B744FD27}
  ```

  Your entry should look like this:

  ```yaml
    authtoken: ${enc:~/.config/geneos/keyfile.aes:+encs+6841BD18C7FB9082742B8208B744FD27}
  ```

* `routingkey` - Required. Default take from environment variable `PAGERDUTY_INTEGRATION_KEY`

  The Pagerduty routing key that sends the event to a specific service. Each service in Pagerduty will have it's own routing key, so depending on how you have defined your Pagerduty services you may, for example, set this as an Geneos Attribute on a Managed Entity or Managed Entity Group, and then it is automatically set with the same name in the environment of the `pagerduty` program when run.

  The Integration Key can be found under the Pagerduty Service -> Integrations tab (select the service under your instance: `https://XXX.pagerduty.com/service-directory`). You should either select the Geneos integration (if it exists in your list) or create a `Custom Event Transformer`. Click on the down arrow to reveal the Integration Key value.

* `alert-type` - default: `${_ALERT_TYPE}`
  The `pagerduty.alert-type` is used (in the code above) to indicate a `clear` or `suspend`. Other values are ignored.

* `severity-map` - default: see file `pagerduty.defaults.yaml`

  The configuration settings under `pagerduty.severity-map` maps Geneos severity values (on the left) to values in the PD-CEF list (`info`, `warning`, `error`, `critical`), recognised by Pagerduty with one exception; The value `ok` is used to indicate this is a resolution and should result in a `resolve` event being sent. In this case the PD-CEF severity is then internally set to `info`.

  The default is to map Geneos `critical` and `warning` to Pagerduty `critical` and `warning` levels and Geneos `ok` is treated as above.

  To build more complex mapping logic, for example using the Pagerduty `error` level for some Geneos items, you can use multiple configuration files and refer to them at execution time using the `-c` flag.

* `event`

  The `pagerduty.event` hierarchy sets the various values in the Pagerduty EventsV2 API.

  * `dedup-key` - default: `${_VARIABLEPATH}`

    The deduplication key is set to the full XPath of the data item that triggered the integration. Sometimes you will want to use other values such as a more general containing data item (the Managed Entity, for example) or a combination of Managed Entity Attributes and other values. You can combine input values by simple concatenation, however be aware of how YAML may treat special characters and if in doubt use quotes. e.g.

    ```yaml
        dedup-key: "${LOCATION}-${_MANAGED_ENTITY}-${_DATAVIEW}"
    ```

  * `client` & `client-url` - defaults: `ITRS Geneos` and `https://www.itrsgroup.com`

    The `client` and `client_url` values are shown in the Pagerduty incident as a link, the `client` being the displayed text (prefixed by `View in`) and the `client_url` the destination. These could be used, for example, to link to your internal knowledge base / wiki for a specific service. Remember that you can build these values in the configuration file from multiple environment variables and static text, e.g. if `COMPONENT` and `RUNBOOK` are Geneos Attributes set on a Managed Entity:

    ```yaml
      event:
        client: ${COMPONENT} Runbook
        client_url: ${RUNBOOK}
    ```

  * `payload`

    These fields represent the [Pagerduty Common Event Format](https://support.pagerduty.com/docs/pd-cef) - the way the YAML file is loaded means the names (on the left) are always treated as case insenstive, so `Summary`, `SUMMARY` and `summary` are all the same.

    * `summary` - default `${_RULE} Triggered`

    * `source` - default `${_MANAGED_ENTITY}`

    * `severity` - default `${_SEVERITY}`

    * `timestamp` - default `${_ALERT_TIME}`

      The `timestamp` will reflect the time of the Geneos Alert, but at this time this value is not reflected in the Pagerduty incident raised.

    * `class` - default `${_SAMPLER}`

    * `component` - default `${_NETPROBE_HOST}`

    * `group` - default `${_SAMPLER_GROUP}`

    * `details` - no default, but see `send-env` below

      The details sub-section is used for the PD-CEF `Custom Details` values. Each `name: value` under `details` is placed in the `Custom Details` field. If `send-env` is `true` below then these details are added after the environment and so may override environment variables of the same name.

  * `links` and `images` - defaults: `_KBA_URLS` and none.
  
    A Pagerduty event payload can contain links and image references. While the default links are set to an Geneos Knowledge Base Articles defined on the data item, these values do not appear in the Pagerduty incident interface at this time.

* `send-env` - default: `false`

This can be used as an aid to debugging and if set to `true` will attach all environment variables as `details` in the `payload` above.

## Known Issues

1. At this time the `_ALERT_TIME` mapping to the PD-CEF `timestamp` does not work. We are investigating.
2. `links` and `images` do not appear anywhere in Pagerduty incidents. We are investigating.
