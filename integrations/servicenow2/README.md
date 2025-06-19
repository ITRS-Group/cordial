# Geneos ServiceNow Integration v2

## Introduction

The Geneos ServiceNow integration (Version 2) connects your Gateways to ServiceNow to raise and manage incidents. The integration understands the environment set-up by Gateway Actions and Alert/Effects, transforming them into ServiceNow field values.

The integration is divided into two commands; A _client_, executed by Gateways, and a background _router_. The proxy acts both as a proxy and an additional layer of configuration driven data transformation. There is also a _query_ command to fetch incidents assigned to a specific user.

The _client_ command uses environment variables and command line flags, while the _router_ then accepts the resulting key/value pairs from clients and creates and sends a ServiceNow table data set. The _client_ command has no direct link to the ServiceNow instance.

>[!NOTE]
>Previous versions of this integration also required a wrapper script to perform the initial mapping of Gateway Action/Effect environment variables, but these functions have now all been incorporated into the client command. This version also uses two distinct configuration files for client and proxy.

## Getting Started

The ServiceNow integration is delivered as a single binary. A command argument is used to select _client_, _router_ or _query_ modes. You will need to install the binary (**`servicenow2`**) in a suitable directory so that it can be executed by Geneos Gateways as a client and, if the same host is used to connect to your ServiceNow instance, run as a proxy.

If your ServiceNow instance can only be contacted from a specific network endpoint then you must also install the binary there and ensure you select a listening port that the Geneos Gateway's ServiceNow _client_ process can connect to.

The example configuration files provided with the integration should serve as a good starting point to get you up and running. You will need to, at minimum, edit the **`proxy`** configuration file with the details of your ServiceNow instance; the network address and authentication details. If your proxy process runs on a different endpoint to your client(s) then you will also need to modify the listening address, which defaults to localhost only (and also consider implementing TLS which requires a certificate and private key matching the host and domain name of the proxy endpoint).

### Getting Started - Run The Proxy

Copy the **`servicenow2`** binary to a suitable directory. If you are not sure where then use either `/usr/local/bin/` (if you have superuser privileges), or `${HOME}/bin/` if you are doing this as a normal user. If `${HOME}/bin` does not exist then create it with `mkdir ${HOME}/bin`. (Note: `${HOME}` is your user home directory and is automatically set by your Linux shell when you login). Check that the binary is executable using `ls -l` or just set it to executable by running `chmod +x [PATH]` where `[PATH]` is the full path to the binary, e.g. `${HOME}/bin/servicenow2`

Create a proxy configuration file using the example one provided with the binary. This configuration file should go in one of three locations that are checked on start-up (in the order given, first one found is used):

* `./servicenow2.proxy.yaml`
* `${HOME}/.config/geneos/servicenow2.proxy.yaml`
* `/etc/geneos/servicenow2.proxy.yaml`

In most cases you will select the first option (the current working directory) for a Gateway if there is a different configuration for each Gateway, or the second for the user running the Geneos Gateways on the server. You may need to create the user config directory with `mkdir -p ${HOME}/.config/geneos` first. So, for example:

```bash
mkdir -p ${HOME}/.config/geneos
cp servicenow2.proxy.example.yaml ${HOME}/.config/geneos/servicenow2.proxy.yaml
```

Edit the new proxy configuration file, following the suggestions in the comments. At minimum you will need to set the ServiceNow instance name and authentication details for the ServiceNow user that will be used to create and update incidents.

First, start by running the proxy in the foreground, so you can watch for any errors:

```bash
servicenow2 proxy
```

If the directory you installed the binary is not in your executable path (such as if you had just created the `${HOME}/bin` directory and not logged out and in again) then use the full path to the binary.

In another terminal session test the proxy by issuing a `curl` command to query existing incidents. To do this you will need the plaintext value of the proxy's authentication token (in the `proxy.authentication.token` configuration field). Then run:

```bash
curl -H 'Authorization: Bearer EXAMPLE' http://localhost:3000/snow/api/v2/incident
```

>[!NOTE]
> If you have changed the listening address or port or the API endpoint path then you must, of course, adjust the URL above to match.

Where `EXAMPLE` should be replaced by your token, in plaintext. You should see output in JSON format with all the configured user's incidents. If you see an empty list (`[]`) then that may be OK. On the proxy side you should see a corresponding log entry, like this:

```log
2025-05-07T15:52:47+01:00  HTTP/1.1 200 0/5524 1.794s 127.0.0.1 GET /snow/api/v2/incident ""
```

Any other results means you need to review the configuration, the ServiceNow user details and the connectivity to your ServiceNow instance.

### Getting Started - Configure The Client

Now that the proxy is running you can build the client configuration file and the Gateway Actions/Effects you want to use it.

Just like for the proxy functionality, the client looks for a configuration file in this order:

* `./servicenow2.client.yaml`
* `${HOME}/.config/geneos/servicenow2.client.yaml`
* `/etc/geneos/servicenow2.client.yaml`
  
If you want to use a different location or filename you can use the `--config /path` option on the command line. For typical use with a Gateway you can put the configuration file in the Gateway working directory (the first in the list above).

For the initial implementation you should start with the example configuration file packages with the binary and edit it as suggested by the comments.

## How It Works

When the Gateway runs an Action from a Rule or an Effect from an Alerting hierarchy, it sets environment variables for the external process to use as a context. The names and values varies slightly between the two and are documented in these sections of the reference manuals:

* [*Geneos Actions*](https://docs.itrsgroup.com/docs/geneos/current/processing/monitoring-and-alerts/geneos_rulesactionsalerts_tr/index.html#action-configuration)

* [*Geneos Effects*](https://docs.itrsgroup.com/docs/geneos/current/processing/monitoring-and-alerts/geneos_rulesactionsalerts_tr/index.html#effects)

In addition to environment variables that Actions or Effects set for specific data-items, variables for the values in each cell on the same row (where the Action/Effect is for a table cell) and also for each Managed Entity Attribute are set.

The servicenow2 client can use these environment variables both to set incident field values and also to test for conditions and set or remove fields driven by these tests.

### Data Flow

```mermaid
---
title: ITRS Geneos to ServiceNow
---
sequenceDiagram
    participant G as Geneos Gateway
    participant C as 'servicenow2 client' command
    participant R as 'servicenow2 proxy' daemon
    participant S as ServiceNow API

    G ->> C: Action/Effect Data<br/>(Environment Variables)
    C ->> R: Incident field updates

    R ->> S: user and cmdb_ci Lookup
    S ->> R: user sys_id and cmbd_ci sys_id returned,<br/> or use configured default

    R ->> S: Lookup incident<br/>(using cmdb_ci and correlation_id)
    S ->> R: Incident sys_id<br/>(or none)

    alt No existing incident
    R ->> S: Create Incident<br/>(fields adjusted for "New Incident")
    else Update incident
    R ->> S: Update Incident<br/>(fields adjusted for "Update Incident")
    end

    R ->> C: Incident identifier<br/>(or error response)
    C ->> G: Return Status
```

### Query Command

There is also a `query` command that will fetch the incidents for the given user. If no user is specified on the command line then the user the proxy uses to connect to the ServiceNow instance will be used. The table to query, which defaults to `incident`, can also be set on the command line. Finally, the output format can be in JSON or a CSV table, suitable for Geneos Toolkit samplers.

The `query` command uses the `servicenow2.client.yaml` configuration file for proxy connection information. The fields that are returned, and the query sent to ServiceNow, are both defined in the proxy configuration and cannot be controlled by the `query` command.

## Client Configuration

The client configuration file controls the transformation of Geneos Action/Effect environment variables to ServiceNow files in the form of name/value pairs. The file is in YAML format but supports Cordial's "expandable" format for almost all values (right of the `:`). See below for more information.

The configuration file is evaluated for each execution, so changes to the file will take effect on the next run.

### Geneos Environment Variables To ServiceNow Fields

All data values are passed from the Geneos Gateway to the integration as environment variables. These are then transformed into the name/value fields passed to the proxy process, which in turn will process them and pass them to ServiceNow to either create or update an incident.

The client configuration includes features to test, set and unset ServiceNow fields based on environment variables.

>[!NOTE]
>The configuration is in the YAML format, so it is important to use the correct indentation. Please pay attention to change you make to ensure these are correct.

This processing is done in two _sections_, `defaults` and a selected _profile_, which are in turn made up of _action groups_. First the [**`defaults`**](#defaults) section is evaluated and then the selected profile section. Each section is processed as an ordered list of groups of tests and actions.

Each _action group_ supports the following actions (more details [below](#actions-groups)):

* `if` - Continue processing this group if the value(s) evaluate to `true`. The default is to act as if `if: true` is used.
* `set` - A list of key/value pairs. Evaluates the right side and sets the named field, overwriting any previous value
* `unset` - Removes the field from the currently defined set
* `subgroup` - Starts a new, lower level, group which is then processed in order and recursively
* `break` - Stop further processing of the **_containing_** group

The order that action are defined in a group is not important as they are always processed in the order above.

### ServiceNow Fields

All the fields built on the client-side are passed to the proxy process, which will perform further processing. Those stages are described further below, but it is worth noting the following behaviour; All fields that start with an underscore (`_`) are internal to the integration and are discarded by the proxy after processing and are used to pass values that control that processing. For example, the `_subject` field can be used as the `short_description` when creating a new incident or included in the `work_notes` when updating an existing incident (or dropped entirely). Other internal fields may be simply to pass query information, such as `_cmdb_ci_default`.

Internal fields with special meanings are:

* `_subject` - Default `Geneos Alert on ${select:_NETPROBE_HOST:Unknown} | ${select:_DATAVIEW:None} | ${_ROWNAME} | ${_COLUMN}${_HEADLINE} is ${_VALUE}`

  This field is used to populate the `short_description` field when creating new incidents and discarded otherwise.

* `_text` - Default is a multiline summary of the alert

  This field is used to populate the `description` field for new incidents and the `work_notes` field for incident updates.

* `_cmdb_ci_default` - No default

  The `sys_id` value for the default configuration item if the search returns nothing. For example, defining a Managed Entity Attribute with a known `sys_id` and then using that like:

  ```yaml
  defaults:
    - set:
      _cmdb_ci_default: ${SNOW_SYS_ID}
  ```

* `_cmdb_search` - Default `name=${_NETPROBE_HOST}`

  The Service Now formatted search string to lookup the CMDB_CI. The supported format is documented across these pages:

  * <https://www.servicenow.com/docs/bundle/xanadu-platform-user-interface/page/use/using-lists/concept/c_EncodedQueryStrings.html>
  * <https://www.servicenow.com/docs/bundle/xanadu-platform-user-interface/page/use/common-ui-elements/reference/r_OpAvailableFiltersQueries.html>

* `_cmdb_table` - Default `cmdb_ci`

  The name of the table to search for the configuration item using `_cmdb_search` above.

* `_table` - Default `incident`

  The name of the incident table to use. This can be overridden by the command line `--table/-t` flags.

* `_profile` - Default `default`

  The name of the Action Group profile to use. This can be overridden by the command line `--profile/-p` flags.

* `_correlation_id` - Default `${_GATEWAY}${_NETPROBE_HOST}${_MANAGED_ENTITY}${_SAMPLER}${_DATAVIEW}${select:_ROWNAME+$_COLUMN:_HEADLINE:}`

  This is the textual correlation string used to generate the real `correlation_id` field that is used to search for an existing incident. The `correlation_id` field is formed from the SHA1 checksum/hash of the given string. If a `correlation_id` field is defined then this internal field is not used.

* `_update_only` - Default `false`

  If set to true, then incidents are only updated if they already exist. This can be useful where your monitoring should only update an existing incident, for example if a WARNING severity event is triggered that should only update an existing CRITICAL, but not raise it's own incident.

### Value Expansion

Almost all the right-hand side values in the YAML configuration file support a custom expansion syntax. Expansion is not recursive, unless a customer function mentions it, but multiple expansion functions can be given in one entry and are concatenated as if they were all one string value, e.g.:

```yaml
    _subject: Geneos Alert on ${select:_NETPROBE_HOST:Unknown} | ${select:_DATAVIEW:None} | ${_ROWNAME} ${_COLUMN}
```

The standard _cordial_ expansion functions:

* `${env:ENV}` or `${ENV}`

  Return the value of the environment variable `ENV`. The second, shorter, format takes precedence of the config option below as `ENV` will very rarely contain a dot. In those very rare cases where environment variable names contain a dot character use the first format. If the environment variable does not exist then an empty string is substituted and no error is logged.

* `${config:ITEM.ITEM...}` or `${ITEM.ITEM...}`

  Return the value of the configuration item `ITEM.ITEM` etc. The second form only works when there is at least one dot separator in the configuration item path. This can be used to pull in the value of another configuration key (which can be done in native YAML, but this allows for clearer configurations). The item referred to is not expanded.

* `${file:/path/to/file}`

  Substitute the contents of the file given by the path. If the path starts `~/` then this is relative to the home directory of the user.

* `${http://example.com/path}` or `${https:///example.com/path}`

  Substitute the contents of the remote URL.

* `${enc:/path/to/aesfile:ENCRYPTED}`

  Decrypt the `ENCRYPTED` string using the AES file path given. Use this to embed credentials which cannot be decrypted without the AES file. To create a field of this format you can run `geneos aes password`.

Additional custom functions:

* `${match:ENV:PATTERN}` - evaluate PATTERN as a regular expression against the contents of `ENV` environment variable and return `true` or `false`. If ENV is an empty string (or not set) then `matchenv` returns `false`

* `${replace:ENV:/PATTERN/TEXT/}` - apply the regular expression `PATTERN` to the value in the `ENV` environment variable and replace all matches with `TEXT`. `PATTERN` and `TEXT` support the features provided by the Go `regexp` package. To include forward slashes (`/`) or closing brackets (`}`) you must escape them with a backslash.

* `${select:ENV1:ENV2:...:DEFAULT}` - return the value of the first environment variable that is set (including an empty string). The last field is a string returned as a default value if none of the environment variables are set. Remember to include the last colon directly followed by the closing `}` if the default value should be an empty string.

  Each ENV* field may also include multiple environment variable names separated by one of: `-`, ` `, `/`, `+` or `++`. A single plus (`+`) is a zero-width separator while a double plus (`++`) results in a single plus sign in the output. The other valid separators are included as-is. If any of the individual environment variables are set then this counts as if the whole field were set. This feature is useful for alternating between aggregate labels each as rowname and column versus headlines.

  For example:

  When `_ROWNAME` is `row`, `_COLUMN` is `data1` and `_HEADLINE` is `headline1` group:

  * `${select:_ROWNAME+_COLUMN:_HEADLINE:None}` becomes `rowdata1`
  * `${select:_ROWNAME/_COLUMN:_HEADLINE:None}` becomes `row/data1`

  And when `_ROWNAME` and `_COLUMN` are unset then:

  * `${select:_ROWNAME/_COLUMN:_HEADLINE:None}` becomes `headline1`

  Other separators are not valid and result in the field being ignored. Separators may appear before or after the environment variable names as well, but are not taken into account when checking if the values are set.

* `${field:FIELD1:FIELD2:...:DEFAULT}` - returns to current value of the first ServiceNow field set. As per `select` above, DEFAULT is a string that is returned if non of the fields are set.

### Client Actions Groups

Action Groups can contain one of each of the named entries below. The entries are evaluated in a specific order, as shown in the introduction above (and in the order of the sections below).

#### `if`

The `if` action evaluates either a single value or an array of values, which must all be true (and so an array of values acts like `AND`-ing the values together). If the test(s) are `true` then the rest of the group is actioned, if `false` then processing of the current group stops and any further groups are then evaluated.

Not providing an `if` is the same as using `if: true`.

The kinds of layout below are supported:

```yaml
defaults:
  # single TEST
  - if: TEST
    set: ...
  # list of TESTs
  - if:
    - TEST1
      TEST2
    set: ...
  # alternative list of TESTs
  - if: [ TEST1, TEST2 ]
    set: ... 
```

#### `set`

```yaml
set:
  key: value
  key: value
```

The `set` action, as the name suggests, sets the fields to the expanded values on the right.

#### `unset`

`unset` removes fields for the current set and can be one or more field names. `unset` only works on existing fields and will not affect further processing which results in the field being re-added.

```yaml
  if: ${match:_SEVERITY:\bok\b}
  unset: _text
```

#### `subgroup`

A `subgroup` introduces group(s) that contain further action group(s) as an array of YAML objects.

For example, these two `if` sections result in identical results:

```yaml
defaults:
  - if: TEST
    set: ...

  - if: TEST
    subgroup:
      - set: ...
```

Using `subgroup` is useful when further tests are required and these are also processed in top down order. For example, grouping together tests that all depends on a _parent_ value, such as testing a Managed Entity Attribute value. e.g.

```yaml
defaults:
  - if: ${match:ENVIRONMENT:\bPROD\b}$
    subgroup:
      - if: ${match:CATEGORY:\bDatabase\b}
        set:
          ...
```

See `break` below as a way of exiting a list of groups based on a test.

#### `break`

`break` evaluates the value(s) on the right, the same way as the `if` action, and terminates further processing of groups in the same parent group or section. Break is, however, evaluated last and so allows other actions to be processed in the same group and then (as the name suggests) break further processing below at the same level.

```yaml
  # long format
  - if: ${match:_SAMPLER:mySampler}
    break: true

  # combined
  - break: ${match:_SAMPLER:mySampler}

  # more actions
  - if: ${match:_SAMPLER:mySampler}
    set:
      _subject: Alert in ${_SAMPLER}
    break: true
```

### Client Configuration Sections

* `proxy`

  The first part of the configuration file is `proxy` and contains the settings on how to communicate with the proxy process.

  * `url`

    The URL of the proxy. This is normally of the form `https://HOSTNAME:PORT/snow/api/v2` where `HOSTNAME` is likely to be `localhost` and the default port is 3000. These values are set in the proxy configuration, and if you are not responsible for configuring the proxy part of this integration then please obtain the correct value from the administrator.

  * `default-user` - Default `admin`

    This is the default ServiceNow user name to use for the `query` command and is ignored for the `client` command.

    The default value of `admin` is only useful for testing and should be set carefully.

  * `authentication`

    The `authentication` section needs to match the settings in the proxy. At this time only `token` is supported.

    * `token`

      A string which must match the setting on the proxy. This value can be opaqued by using a _cordial_ AES encoded expansion value, such as that generated by `geneos aes password`.

  * `tls`

    If the `url` above uses a `https` scheme then the settings in this section take effect. They are ignored otherwise.

    * `skip-verify` - Default `false`

      By default the proxy certificate must both match the hostname portion or the URL and the certificate chain is also checked. If `true` then no certificate validation is done.

    * `chain`

      If set, `chain` is used to verify the proxy certificate. The chain should be one or more PEM encoded certificates and can either be given inline (for example as a multiline YAML value) or loaded from a file using `${file:/path/to/file.pem}` style expansion values.

      When no chain is given, then only the operating system PKI CA certificates are used for verification. Note that these are always used, even if a custom `chain` is set.

* `defaults`

  This section sets default values for fields. It is made up of a list of [Action Groups](#client-actions-groups) that are processed in the order given.

* `profiles`

  When the Geneos Gateway runs the integration client it can select a _profile_. If no profile is selected then a profile labelled `default` is used. Note that this is different to the top-level `defaults` above. Using profiles allows you to reduce the required nesting of tests  by categorising into related action groups, such a `opsview` or `infrastructure` and then using this name from different Geneos Actions or Alert/Effects in the Gateway.

  Each profile is made up of a list of [Action Groups](#client-actions-groups) and is processed in order.

## Proxy Configuration Reference

### Proxy Configuration Sections

* `server`

  * `listen`

  * `path`

  * `tls`

    * `enabled`

    * `certificate`

    * `private-key`

  * `authentication`

    * `token`

* `servicenow`

  * `url`

  * `path`

  * `username`

  * `password`

  * `client-id`

  * `client-secret`

* `tables`

  Incidents can be applied to different tables, depending on the configuration. By default only the `incident` table is configured. The `tables` section can include a list of tables, each containing the keys below.

  * `name` - No default

    The name of the table in ServiceNow, e.g. `incident`

  * `search`

    `search` is the ServiceNow filter query used to lookup existing incidents

  * `response`

    This section defines the three messages returned to the caller on incident creation, update or failure. It can be used to format custom messages which include values of interest to your specific installation. Expansion values include fields by name after all other processing is complete but before internal fields are removed.

  * `defaults`

    The `defaults` section contains key/value pairs of field values that are used as defaults if they are not set by the client or by processing (see below) in the proxy.

  * `current-state`

    The `current-states` section is evaluated based on the existing numeric value of the state of the incident in ServiceNow, and using `0` if no existing incident is found.

    Each state-valued block can include the following (evaluated in the the order shown):

    * `defaults`

      `defaults` sets default values for the given current state. These override the general defaults above but only take effect if the fields are not defined by the client.

    * `remove`

      A list of fields to remove.

      Note that all internal fields, those prefixed with an underscore (`_`) are removed after the evaluation done in `current-states`.

    * `rename`

      A list of old and new names for fields to rename. This is the configuration section where internal fields, like `_text` can be set to their final names, like `work_notes`

    * `must-include`

      A list of fields that must be present in the set at the point of evaluation, or an error is returned to the caller.

    * `filter`

      If defined, a list of regular expression filters that are applied to the ServiceNow fields, and fields not matching at least one of the filters is removed from the set to ServiceNow. If there is no filter list defined then no filtering is performed.

  * `query`

    The `query` section controls how the proxy handles queries to the API endpoint.

    * `enabled` - Default `false`

      This should be set to `true` to enable the query endpoint.

    * `fields` - No default

      A list of fields names to fetch for the query.

