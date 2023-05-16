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
  $ cd $(geneos home MyGateway)
  $ _VARIABLEPATH='//dataview[(@name="SOMEDATAVIEWNAME")]' dv2email
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
  * `port` - default `7038`
  * `use-tls` - default `true`
  * `allow-insecure` - default `true`
  * `name` - no default
  * `username` - no default
  * `password` - no default

* `email`

  * `smtp` - default `localhost`
  * `port` - default `25`
  * `username` - no default
  * `password` - no default
  * `from` - no default
  * `to` - no default
  * `subject` - default `Geneos Alert`

* `column-filter` - default from Environment Variable `__columns` (two underscores)
* `row-filter` - default from Environment Variable `__rows` (two underscores)
* `headline-filter` - default from Environment Variable `__headlines` (two underscores)
* `first-column` - default from Environment Variable `_FIRSTCOLUMN`
* `row-order` - default first column ascending

  These five configuration settings influence the way that Dataview cells are passed into the templates.


* `images`

  A list of image files to embed into the resulting email. The name (on
  the left) is used as the href `cid` value.

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
