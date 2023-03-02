# A Drop in Replacement for Geneos `libemail.so`

`libemail.so` is intended as a drop-in replacement for standard `libemail.so` with the following extras:

* For e-mail notifications
  * Enhanced, modern SMTP support
  * TLS
  * Authentication
  * Go templates for both text and HTML
  * HTML and CSS support
  * EMail meta parameters are removed from list available to formats
* For other notifications
  * Added function to send a notification message to a msTeams channel

## Building

If you do not download a ready compiled binary then you can build from source.

You must have Go 1.17 or later installed as well as `make` and any other tools required by the CGo toolchain.

```bash
git clone https://github.com/itrs-group/cordial.git
cd cordial/libraries/libemail
make
```

If you do not have `make` installed you can build using:

```bash
go build -buildmode c-shared -o libemail.so *.go
```

Then copy the resulting `libemail.so` to a suitable location and add the path to the Gateway configuration. You can replace the `libemail.so` file in the official distribution but you should probably backup the original file by renaming it:

```bash
mv libemail.so libemail.so.orig
```

Note that the Gateway does not reload any shared libraries it has already loaded and so a Gateway restart may be required to pick up the new library.

## Using

### As a replacement for `libemail.so`

The official `libemail.so` is [documented here](https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_rulesactionsalerts_tr.html#Libemail) and all of the parameters are supported.

Note that if you are using this as a literal drop-in replacement you may need to restart the Gateway process to load the new library, if the old one was already loaded from the same location.

#### New features

The following additional parameters are supported by the `SendMail` function:

* `_SMTP_USERNAME`
* `_SMTP_PASSWORD`
* `_SMTP_PASSWORD_FILE` **Deprecated**
* `_SMTP_TLS`

If `_SMTP_USERNAME` is set then authentication is attempted. As the Gateway writes all parameters to it's log file, the password should either be encoded using [ExpandString](https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#Config.ExpandString) or placed in a file with restrictive permissions that can be read by the Gateway process. The `_SMTP_PASSWORD_FILE` is either and absolute path or a path to a file relative to the working directory of the Gateway. If defined then `_SMTP_PASSWORD` overrides the value in any file referenced by `_SMTP_PASSWORD_FILE`.

To use an encoded password it must be in the format `${enc:KEYFILE:CIPHERTEXT}`. This is the format supported by [ExpandString](https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#Config.ExpandString) and can be created using the [`geneos`](/tools/geneos/README.md) program or manually by following the instructions in [Secure Passwords](https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm). Assuming you have created a keyfile and saved it to the default location (e.g. `geneos aes new -k ~/.config/geneos/keyfile.aes`) then you can generate the value like this:

```bash
# first, create a keyfile, if one does not exist
$ geneos aes new -D
$ geneos aes encode -e
Password: [ENTER PASSWORD HERE]
${enc:~/.config/geneos/keyfile.aes:+encs+02554ABDB5474D9FC604CAF63E50918C}
```

Copy the entire output line to the value for `_SMTP_PASSWORD` and ensure that the keyfile is accessible to the Gateway process. You can copy the keyfile to a new location, such as the Gateway's working directory, but remember to update the path in the `${enc:...}` setting.

Note: Another benefit of using ExpandString encoding is that the Gateway automatically redacts any string that starts `+encs+` in it's output, so all the log will show is (literally) `_SMTP_PASSWORD=${enc:~/.config/geneos/keyfile.aes:XXX}` which can serve as an extra layer of protection against casual viewing of the logs and configuration files.

`_SMTP_TLS` can be one of `default`, `force` or `none` (case insensitive). `default` is to try TLS but fall back to plain text depending on the SMTP server.

If you use this library with authentication and connect to a public server, such as GMail or Outlook, then you should always create and use a unique "app password" and never use the real password for the sender account.

### `GoSendMail` function

This is a forward compatible function that accepts almost the standard parameters (except those with `FORMAT` in the name) above but will also add an HTML part to the EMail.

#### Go templates

The following new parameters are used to support Go templates - both text and HTML:

* `_TEMPLATE_TEXT`

  Override the built-in text template. This is now a single block of configurable text and uses Go templates to embed the logic to evaluate different Alert types that was previously performed in code and with multiple formats.

* `_TEMPLATE_TEXT_FILE`

  Override the build-in text template with the contents of the named file. This takes precedence over `_TEMPLATE_TEXT`

* `_TEMPLATE_TEXT_ONLY`

  If this is set (to any value) then the function will send a text-only EMail and not process any HTML or CSS settings

* `_TEMPLATE_HTML` and `_TEMPLATE_HTML_FILE`

  Similar to the above, these settings override the default HTML template. The default HTML template is almost identical to the text one except all parameter values are rendered in *bold*.

* `_TEMPLATE_HTML_ONLY`

  **Not yet implemented** This option will send HTML only email and avoids a multipart MIME message.

* `_TEMPLATE_CSS` and `_TEMPLATE_CSS_FILE`

  Similar to the above, these settings override the default CSS template. The CSS template is included in the HTML template, whether default or user defined, using the following syntax and should be enclosed in `<style type="text/css>...</style>` tags:

  ```css
  {{template "css"}}
  ```

* `_TEMPLATE_LOGO_FILE`

  Override the default embedded logo, which is a Material notification icon. This should be a PNG file and is referenced in the HTML as:

  ```html
  <img src="cid:logo.png" />
  ```

Note: If you use any of the non-FILE settings then the Gateway will log the full template text in the log each time the `GoSendMail` function is invoked. This may result is very large log lines. It is suggested you use the `_FILE` suffixed settings for anything other than very simple templates.

In a future version it is expected that multiple files will be loadable using Go's embed FS features.

### Debug

There is one built in `_DEBUG` parameter but you can also add your own to the template logic and `_TEMPLATE_DEBUG` has been included in the built-in templates to demonstrate this.

* `_DEBUG`

  If set to `true` (case insensitive) prevents EMail meta parameters (e.g. `_FROM`, `_SMTP_SERVER` etc.) from being removed from the parameters passed to formats or templates. This includes the plain text password, if provided, so beware. You can then output these values in your custom formats and templates for review.

* `_TEMPLATE_DEBUG`

  This example parameter outputs a text and HTML table of all parameters, unsorted, which may or may not include the EMail meta-parameters, depending on `_DEBUG` above. In the built-in templates this has to be either `TRUE` or `true` and will not work for `True`, for example.


## `GoSendToMsTeamsChannel` function

This is a function that sends notification messages to one or multiple Microsoft Teams channels, using the incoming webhook API.
A pre-requisite for using this function is to create an incoming webhook for each target channel in Microsoft Teams.  See following refs for details:
  * https://docs.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook
  * https://techcommunity.microsoft.com/t5/microsoft-365-pnp-blog/how-to-configure-and-use-incoming-webhooks-in-microsoft-teams/ba-p/2051118
  * https://docs.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/connectors-using?tabs=cURL

The following parameters are used to send messages to Microsoft Teams chanela with support for `Go` templates (both text & HTML):
* `_TO`
List of Microsoft Teams incoming webhook URLs, separated by `|` (pipe) character.
* `_SUBJECT`
Similar to the legacy parameter used by the `SendMail` function.
This parameter supports both the `Go` template & the legacy Geneos formatting.
* `_TEMPLATE_HTML_FILE`
Override the built-in template (default) with the contents of the `Go` HTML template file whose path is defined in this parameter.
This takes precedence over `_TEMPLATE_HTML`, `_TEMPLATE_TEXT_FILE`, `_TEMPLATE_TEXT` & `_FORMAT`.
* `_TEMPLATE_HTML`
Override the built-in template (default) with a single block of configurable text using a `Go` HTML template format.
This takes precedence over `_TEMPLATE_TEXT_FILE`, `_TEMPLATE_TEXT` & `_FORMAT`.
* `_TEMPLATE_TEXT_FILE`
Override the built-in template (default) with the contents of the `Go` text template file whose path is defined in this parameter.
This takes precedence over `_TEMPLATE_TEXT` & `_FORMAT`.
* `_TEMPLATE_TEXT`
Override the built-in template (default) with a single block of configurable text using a `Go` text template format.
This takes precedence over `_FORMAT`.
* `_FORMAT`
Override the built-in template (default) with a single block of configurable text using either a `Go` template format or a Geneos legacy format.
