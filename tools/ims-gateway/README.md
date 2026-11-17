# Incident Management System (IMS) Gateway

## Overview

The Incident Management System (IMS) Gateway is a component of the Geneos ecosystem that provides a flexible and extensible way to integrate with various incident management systems. It acts as a bridge between Geneos and the target IMS platforms, allowing you to create and update incidents based on alerts generated in Geneos. The IMS Gateway is designed to be highly configurable, allowing you to control the behaviour of the incident creation and update process through the use of special fields in the data that is sent to the IMS Gateway.

The client side is currently provided by the `geneos incident` commands, which can be used in rules and actions to send data to the IMS Gateway. The IMS Gateway can then process this data and create or update incidents in the target IMS platform based on the configuration and the special fields provided in the data. The configuration of the client-side is through a separate configuration file which is documented ...

Currently ServiceNow and ServiceDesk Plus are supported as target IMS platforms, but the IMS Gateway is designed to be extensible and can be easily extended to support additional platforms in the future. The IMS Gateway is implemented in Go and uses a plugin architecture to allow for easy addition of new target platforms without requiring changes to the core IMS Gateway code.


## Running In The Background

Run the gateway in the foreground with `ims-gateway start`, or in the background with the `--daemon` (or `-D`) flag, and a log file:

```bash
ims-gateway start --daemon --logfile /var/log/geneos/ims-gateway.log
```

The command only returns once the gateway is listening, and prints its PID. If the gateway fails to start, for example if the configuration is invalid or the port is in use, then the command fails with a non-zero exit code and the reason is in the log file.

While it runs, the gateway holds a PID file, and a second gateway cannot be started with the same one. The file is the one given with `--pidfile`, or `server.pidfile` in the gateway configuration, or if neither a file in the `geneos` directory in your user cache directory. If you run more than one gateway on a server then give each its own. The PID file also records how the gateway was started.

Use these commands to control a gateway running in the background. Give them the same `--conf` or `--pidfile` options if you started the gateway with them, or if it uses a configuration file that is not in the default locations:

* `ims-gateway status` - show whether the gateway is running. The exit code is zero only if it is.
* `ims-gateway stop` - stop the gateway. It stops accepting requests and lets those in progress complete, for up to 15 seconds, before exiting. The command waits for this, and fails after `--timeout` unless `--force` is used, which kills the process.
* `ims-gateway restart` - stop the gateway, as above, and start it again with the same arguments. Use this to run a new version of the program after an upgrade.
* `ims-gateway reload` - make the gateway read its configuration again, without stopping.

`reload` checks the configuration first, and only tells the gateway to reload if it is valid, so a mistake in the file is reported straight away. The gateway checks it again, and if it fails the gateway carries on with the configuration it has and logs the error. A reload changes everything in the configuration that the gateway uses to handle requests, such as the authentication token, the ServiceNow connection details and the tables, and a changed TLS certificate and key. If the listening address is changed then the gateway listens on the new address before it stops listening on the old one. Requests that are in progress complete using the configuration that they started with, and all new requests use the new one. Changes to the logging, and turning TLS on or off, need a `restart`. You can also reload by sending the gateway process the `SIGUSR1` signal. Reload is not available on Windows.

## ServiceDesk Plus Authentication

The IMS Gateway supports authentication against ServiceDesk Plus using OAuth2 Client Credentials Grant. This allows you to securely authenticate with ServiceDesk Plus and obtain access tokens that can be used to create and update incidents in ServiceDesk Plus. The authentication process involves obtaining a `client_id` and `client_secret` from the ServiceDesk Plus admin console, generating a grant token, and then using that grant token to obtain access and refresh tokens that are stored securely in the user's configuration directory. The IMS Gateway will automatically handle token refreshes as needed, so you don't have to worry about manually refreshing tokens. For more details on how to set up and use ServiceDesk Plus authentication, please refer to the [ServiceDesk Plus OAuth2 Authentication](SDP-AUTH.md) documentation.


## Fields

Any fields passed by clients to the IMS Gateway that have a prefix of two underscores (`__`) will be treated as special fields and will be used to control the behaviour of the IMS Gateway or as general metadata that can be converted to platform specific values. Once the IMS Gateway has processed the incoming data, these fields will be removed from the data that is sent to the target system to avoid any potential conflicts with reserved field names in the target system.

After the prefix an additional identifier can be used to highlight the functional group of the field. For example, `__snow_` and `__sdp_` are used to identify fields that are specific to ServiceNow and ServiceDesk Plus respectively.

## Reserved Field Names

### Incident

All fields with the prefix `__incident_` will be used to create an incident in the target system. The content of these fields will be used as the subject and body of the incident. The correlation field can be used to link related incidents together.

* `__incident_subject`

    The subject of the incident.

* `__incident_body_text`

    The body of the incident in plain text format.

* `__incident_body_html`

    The body of the incident in HTML format. This can be used to provide a richer format for the incident description, including tables and other formatting for those IMS platforms that support it. There should always be a plain text version of the body provided in `__incident_body_text` as well for platforms that do not support HTML or for use in notifications and other contexts where a plain text version is required.

* `__incident_correlation`

    A value that is used to match existing incidents in the target system. This can be used to link related incidents together or to update an existing incident instead of creating a new one. The exact behaviour will depend on the target system and how it handles correlation values. The correlation value should be unique enough to avoid collisions with unrelated incidents but consistent enough to ensure that related incidents are linked together.

    In general the correlation value is hashed by the IMS Gateway before being sent to the target system to ensure that it is of a consistent format and length. The hashing algorithm used is SHA-256, which produces a 64 character hexadecimal string.

* `__incident_update_only`

    If set to `true`, the IMS Gateway will only update existing tickets that match the correlation value provided in `__incident_correlation`. If no matching ticket is found, no new ticket will be created. This should be set, for example, when resolving an incident in Geneos and you want to ensure that the corresponding ticket in ServiceDesk Plus is also resolved without creating a new ticket.

### ITRS

All fields with the prefix `__itrs_` are reserved for raw values that are passed in from the ITRS component invoking the IMS.

These fields are not Geneos specific and could also be derived from Opsview and other ITRS products. They are included here for completeness and to allow for the possibility of using them in the future to control the behaviour of the IMS Gateway or as general metadata that can be converted to platform specific values.

Examples of these fields include:

* `__itrs_gateway`: The name of the gateway that is invoking the IMS.
* `__itrs_netprobe_host`: The name of the NetProbe host that is associated with the alert.
* `__itrs_managed_entity`: The name of the managed entity that is associated with the alert.
* `__itrs_sampler`: The name of the sampler that is associated with the alert.
* `__itrs_dataview`: The name of the dataview that is associated with the alert.

### ServiceNow

* `__snow_cmdb_ci`

* `__snow_cmdb_ci_default`

* `__snow_cmdb_search`

* `__snow_cmdb_table`

* `__snow_table`

* `__snow_correlation`

### ServiceDesk Plus

* `__sdp_status`

    The status of the incident in ServiceDesk Plus. This can be used to control the status of the incident when it is created or updated. The exact values that are accepted will depend on the configuration of the ServiceDesk Plus instance, but common values include `Open`, `In Progress`, `Resolved`, and `Closed`.
