# ITRS Geneos to Pagerduty Event Integration

This program allows you to send ITRS Geneos alerts to Pagerduty as Events which can, in turn, raise Incidents.

## Getting Started

You can either download a binary or build from source. The default configuration tries to transform Geneos Alert / Action values into a sensible mapping into a Pagerduty event. You will need to supply an authentication token and an integration ID.

## Configuration

The integration takes it settings from the following, in order of priority from highest to lowest:

1. Command line flags
2. Configuration file
3. External Defaults File
4. Internal Defaults

Geneos passes alert information to external programs and scripts using environment variables. These are used by the configuration options to build a Pagerduty Event in [PD-CEF](https://support.pagerduty.com/docs/pd-cef) format.

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

* If the sub-command (the `eventType`) is `Resolve` then do just that
* If the Geneos severity, via the configuration file, maps to the value "ok" or the Alert type (`_ALERT_TYPE`) is a "clear" then this is a `resolve`
* If the sub-command is `Assign` or the Alert type is a "suspend" then send an `acknowledge`
* Otherwise this is a `trigger`

For `resolve` and `acknowledge` the severity is set to `info` regardless of other settings.

The one choice that may need clarification is the selection of `acknowledge` events; From the Geneos side an Alert is suspended if the data item (or parent data item) is Snoozed or Assigned or if it becomes inactive (e.g. via an Active Time). These Geneos state changes roughly meet the criteria on the Pagerduty side for [Incident Statuses](https://support.pagerduty.com/docs/incidents) - i.e. once a Geneos data item is snoozed, assigned or inactive then it is either being worked on or not considered immediately significant. If you are using Alerts, as opposed to Actions or Commands, then these settings are reversed automatically when the item is un-snoozed.

Automated assignment/unassignment events depends on the configuration in your Gateway under Authentication -> Advanced -> User Assignment.

### Configuration Logic

All other behaviour is (should be) configurable. When run as either an Action from a Rule or an Effect from an Alert, Geneos sets a number of environment variables which can be used in the configuration to build custom Pagerduty values. The defaults given below (prefixed with an `_` underscore) are some of these. See the Geneos documentation for more details.

The default configuration sets the following:

* `alert-type` - default: `${_ALERT_TYPE}`
  The `pagerduty.alert-type` is used (in the code above) to indicate a `clear` or `suspend`. Other values are ignored.

* `severity-map` - default: see file `pagerduty.defaults.yaml`
  The configuration settings under `pagerduty.severity-map` maps Geneos severity values (on the left) to values in the PD-CEF list (`info`, `warning`, `error`, `critical`), recognised by Pagerduty with one exception; The value `ok` is used to indicate this is a resolution and should result in a `resolve` event being sent. In this case the PD-CEF severity is then internally set to `info`.

* `event`
  The `pagerduty.event` hierarchy sets the various values in the Pagerduty EventsV2 API.

  * `dedup-key` - default: `${_VARIABLEPATH}`
    The deduplication key is set to the full XPath of the data item that triggered the integration. Sometimes you will want to use other values such as a more general containing data item (the Managed Entity, for example) or a combination of Managed Entity Attributes and other values. You can combine input values by simple concatenation, however be aware of how YAML may treat special characters and if in doubt use quotes. e.g.

    ```yaml
        dedup-key: "${LOCATION}-${_MANAGED_ENTITY}-${_DATAVIEW}"
    ```

  * `client` & `client-url`

  * `payload`
    These fields represent the [Pagerduty Common Event Format](https://support.pagerduty.com/docs/pd-cef) - the way the YAML file is loaded means the names (on the left) are always treated as case insenstive, so `Summary`, `SUMMARY` and `summary` are all the same.

    * `summary` - default `${_RULE} Triggered`

    * `source` - default `${_MANAGED_ENTITY}`

    * `severity` - default `${_SEVERITY}`

    * `timestamp` - default `${_ALERT_TIME}`

    * `class` - default `${_SAMPLER}`

    * `component` - default `${_NETPROBE_HOST}`

    * `group` - default `${_SAMPLER_GROUP}`

    * `details` - no default, but see `send-env` below
      The details sub-section is used for the PD-CEF `Custom Details` values. Each entry under details is placed into the custom details section as-is.

  * `links` and `images`

* `send-env` - default: `false`

This debugging aid, if set to `true`, will send all defined environment variables as `details` in the `payload` above.
