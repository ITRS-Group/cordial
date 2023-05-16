# dv2email - Dataview To HTML

`dv2email` is a lightweight program to capture and transform Geneos
Dataviews to HTML suitable for inclusion in email and other messaging
systems.

## Usage

* Configure Your Gateways

  You have to enable the REST Command API in your Gateways. You should
  have a user account on the Gateways that support password
  authentication and is limited to `data` permissions, i.e. a read-only
  account. How to do this can be found in the following documentation:

  * [REST Service 🔗](https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_commands_tr.html#REST_Service)
  * [Authentication 🔗](https://docs.itrsgroup.com/docs/geneos/6.3.0/Gateway_Reference_Guide/geneos_authentication_tr.html)

* Install `dv2email` and `geneos` (optional) somewhere in your execution path (or use the full path to where you install them)

  Next configure your dv2email YAML files:

  * `dv2email.yaml` in the working directory of each Gateway

    This configuration file (see details below) should contain all
    customisations for the Gateway and the format of the emails you want
    to send.

  * `${HOME}/.config/geneos/dv2email.yaml` - where `${HOME}` is the home
     directory of the user running the Gateway(s)

    This configuration file should contain common configuration that
    applies across all Gateways and also email server configuration and
    credentials, as appropriate. If this file contains credentials, even
    when AES256 encrypted, should be only readable by the Gateway user;
    that is `chmod 0400 dv2email.yaml`.

* Store credentials (optional)

  Next, optionally store any credentials in the user's `geneos` managed
  `credentials.yaml` file using `geneos login`. This is not necessary if
  you embed credentials in the dv2email.yaml files.

* Test

  You should now be able to test your configuration. You can do this by
  simulating a minimal alert on the command line:

  ```bash
  cd $(geneos home MyGateway)
  _VARIABLEPATH='//dataview[(@name="SOMEDATAVIEWNAME")]' dv2email
  ```

  Replace `SOMEDATAVIEWNAME` above with the name of a Dataview on your
  Gateway that is likely to match exactly one Dataview. If it matches
  more than one Dataview then in the test template all matching
  Dataviews will be included in the email.

  If the program returns with no output then it has succeeded in sending
  an email, otherwise review the errors and resolve them as needed. Most
  issues will be related to authentication, either to the Gateways or to
  the email server.

* Update Templates

  Update your templates to suite you requirements. The built-in template
  produces output which says clearly that it is for testing and should
  be edited for local requirements. The templates (there is both an HTML
  and a plain text template) embedded into the binary are identical to
  the ones in the example configuration file included alongside the
  binary.

* Use

  If you are happy with the results of your testing then you should be
  able to create and Action or Effect that is simply the program, like
  this:

  ```bash
  dv2email
  ```

  While there are some command line flags you should not need to use
  them in normal operation as all the details are either in the
  configuration file(s) and the environment variables set by Geneos.

## How It Works

The program has been designed to run under a Geneos Gateway and process
the standard environment variables that the Gateway sets when running an
Action from a Rule or an Effect from an Alerting hierarchy. The precise
list of values differs based on which mechanism is used and also the
data item that the Action or Effect is run against. Details are
documented here:

* [Actions 🔗](https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_rulesactionsalerts_tr.html#Action_Configuration)
* [Effects 🔗](https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_rulesactionsalerts_tr.html#Alerting-Effects)

The primary difference for `dv2email` is that when running from an Alert
and Effect some of the email headers are automatically set based on the
Notification set-up. `dv2email` will recognise these and process them as
expected. For Actions the `To`, `From`, `Subject` items must be set. You
can define the defaults in the configuration files and override them
using `userdata` functions in the Rule Block, e.g. (in Geneos):

```text
if value > 100 then
    userdata "_TO" "manager@example.com"
    userdata "_SUBJECT" "HELP! It's Broken!"
    run "dv2email Action"
else
...
```

The contents of the email are assembled from the two templates (text and
HTML) and any image attachments. The resulting HTML is also run through
an "inliner" to ensure that any CSS defined in the HTML `<head>` section
is inlined to the tags that need the settings as many email clients
(GMail and others) only support a simple HTML5/CSS3 format. If you wnt
to save data then you can disable this inlining with the
`--inline-css=false` command line flag.

## Configuration Reference

### File locations

The `dv2email` program will look in three directories for a
`dv2email.yaml` file and merge the contents in the following order (last
version of a value "wins"):

1. `/etc/geneos` - a global configuration directory for `geneos`. These
    are general, global settings and in many instances this directory
    does not exist.
2. `${HOME}/.config/geneos` - the user's `geneos` configuration
   directory. This is the user running the program, which again will be
   typically the Gateway user.
3. Working directory of process - i.e. where you are when you run it,
   not the installation directory (`pwd`). This is normally the same as
   the working directory of the Gateway running it.

Additionally there is support for "defaults" files for all the above.
You can have `dv2email.defaults.yaml` files in any of the above
directories and these are read before the main configurations but after
built-in defaults. They make a good option for complex templates that
would otherwise pollute the visibility of other configuration options.

Note: If the program is renamed then the base name of all the files
above is also changed. e.g. if you rename the program `dv2mailserver`
then the configuration files that the program searches for will be
`dv2mailserver.yaml` etc.

### Configuration Options

The configuration is in three parts; Gateway connectivity, EMail server
connectivity and everything else. These are described below along with
the defaults (except the templates which can be quite large):

* `gateway`

  This section is for the connectivity to the Gateway REST API. If
  `dv2email` is running alongside the Gateway then you probably only
  need to configure the username and password.

  * `host` - default `localhost`

    The hostname or IP address of the Gateway.

  * `port` - default `7038`

    The port that the Gateway accepts REST Commands on. The default is
    7038 regardless of the `use-tls` setting below. This is intentional
    as the REST Command API will normally only accept commands on a
    secure port.

  * `use-tls` - default `true`

    Use a secure connection. This is the default for the REST Command
    API when enabled.

  * `allow-insecure` - default `true`

    This setting controls the checking of the Gateways server
    certificate and default to `true` as most Gateways will use private
    certificates.

    In a future release the program may be able to automatically check
    against the certificate chain created and maintained by the `geneos`
    program.

  * `username` - no default
  * `password` - no default

    The username and password used to authenticate to the Gateway REST
    Command API. The password should normally be AES256 encrypted using
    Geneos formatted secure passwords but enclosed in `cordial`
    expandable format. These can be generated using `geneos aes password`.

  * `name` - no default

    If no username and password are configured then the program tries to
    locate credentials using the value of `name` - typically the gateway
    name - that have been created and stored using `geneos login`. The
    credential used must be prefixed with `gateway:` to the login
    command. e.g.

    ```bash
    geneos login gateway:MyGateway -u readonly
    ```

* `email`

  * `smtp` - default `localhost`

    The hostname or IP of the SMTP server.

  * `port` - default `25`

    The port of the SMTP server. While the default is 25 most modern
    SMTP server will be listening on ports 465 or 587 depending on their
    configured services, especially when using TLS to protect the
    connection.

  * `use-tls` - default `default`

    By default the the SMTP connection is made using opportunistic TLS,
    i.e. TLS is used if the server advertises STARTTLS but otherwise the
    email is sent in the clear. The other options are `force` and `none`
    which do what the names suggest.

    Note that is it not possible to ignore server certificate errors for
    SMTP. This is intentional.

  * `username` - no default
  * `password` - no default

    The username and password to use for the SMTP connection. The
    password should ne AES256 encrypted as for the Gateway password
    above.

    If no username or password are given then the SMTP connection is
    attempted without authentication.

    In a future release there may be support for fetching these values
    from the `cordial` credentials store but for now they must be in one
    of the `dv2email` configuration files.

  * `from` - no default
  * `to` - no default
  * `subject` - default `Geneos Alert`

* `column-filter` - default from Environment Variable `__COLUMNS` (two underscores)
* `row-filter` - default from Environment Variable `__ROWS` (two underscores)
* `headline-filter` - default from Environment Variable `__HEADLINES` (two underscores)
* `first-column` - default from Environment Variable `_FIRSTCOLUMN` (single underscore)
* `row-order` - default first column ascending

  These five configuration settings influence the way that Dataview
  cells are passed into the templates.
  
  The three `filter` items all work the same way but have some
  difference depending on the dimension of data they apply to. The
  configuration formats all follow the same pattern:

  ```yaml
  column-filter:
    pattern1: [ item1, item2, item3 ]
    pattern2: [ item4, item5, item6 ]
    '*': [ other, values ]
  ```

  The pattern on the left is matched against the Dataview name and for
  all the patterns that match the longest match is selected. This means
  you can have specific configuration for one Dataview and then more
  general defaults for others. The pattern matching is not a regular
  expression but the simpler shell style file patterns known as
  `globbing`. The supported patterns are documented in the Go
  [path.Match 🔗](https://pkg.go.dev/path#Match) docs. The final pattern
  above, the catch-all wildcard must be enclosed in quotes for YAML to
  be valid.

  Once the list of items is matched they are then applied to the data
  set in the following ways:

  * rows - each item is matched against the rowname using the same
    `globbing` rules as above. The total set of rows matched is passed
    to the template in the `Rows` slice. The order of `Rows` is further
    refined by the `row-order` item (see below).

  * columns - each item is matched against the columns names (except the
    first column, see below) and the order of the columns is determined
    by how they matched the items.

    The first column, the `rowname`, is special and is always included.
    If the program is called from the Gateway on a Dataview table cell
    then the environment variable `_FIRSTCOLUMN` is set and this is used
    instead of the literal `rowname`. The configuration item
    `first-column` can be used, with the same syntax as for the filters
    above, to define the name on a per-Dataview basis.

  * headlines - Headline cells are treated in a similar way to columns
    and for all the patterns that match the Dataview name, each item is
    matched against all the available headlines and all that match are
    passed into the template. Headlines are not ordered in anyway.

  Rows can be ordered by one column, including the name of the first
  column (or `rowname` if none is defined) using a similar pattern match
  to the filters above. Only the first item is used and it must be an
  exact match for a column name followed by an option '+' or '-' to
  indicate ascending or descending order, respectively.

* `images`

  A list of image files to embed into the resulting email. The name (on
  the left) is used as the href `cid` value. e.g.

  ```yaml
  images:
    logo1.png: /path/to/my/logo.png
    alert.png: /path/to/another/image.png
  ```

  In a future release it may be possible to refer to images using URLs
  or other "expandable" formats but for now they must be file paths and
  if relative they must be accessibkle from the working directory of the
  process.

* `text-template`

  A template in Go [text/template](https://pkg.go.dev/text/template)
  format to be used to generate the plain text to be used as the
  `text/plain` alternative part in the email. This part of the email is
  not normally visible in modern email clients but it is used for
  assistive text readers and other accessibility tools and should be used
  to describe the contents of the email.

* `html-template`

  A template in Go [html/template](https://pkg.go.dev/html/template)
  format to be used to generate the HTML to be used as the `text/html`
  alternative part of the email.

  The data available to the template (and the text template above) is
  details in the `dv2email.yaml` file.

  For both template types it is possible to include the contents of a
  file or a URL using "expandable" syntax, like this:

  ```yaml
  text-template: ${https://myserver.example.com/files/txt.gotmpl}
  html-template: ${file:/path/to/template.gotmpl}
  ```
