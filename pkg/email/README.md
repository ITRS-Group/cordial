# `email` Package

The `email` package supports ITRS Geneos specific EMail parameters as
documented in the official
[`libemail`🔗](https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_rulesactionsalerts_tr.html#Libemail)
but with additional support for TLS and Basic Authentication.

This package duplicates the original functionality in the `cordial`
[`libemail`](/libraries/libemail/README.md) library but for more general
use.

The Dial method accepts a [`config`](/pkg/config/README.md) object and returns a [`mail.Dialer`](https://pkg.go.dev/github.com/go-mail/mail/v2) or an error.

The new parameters, which are case-insensitive, are:

* `_smtp_username` - no default

    If set then authentication is attempted, using the strongest
    mechanism offered by the server unless `_smtp_auth` is set.

* `_smtp_password` - no default

    The password is only used if `_smtp_username` is defined. It should
    be stored encrypted using the "expandable" format created using [`geneos aes
    password`](/tools/geneos/docs/geneos_aes_password.md) program

* `_smtp_tls` - default `default`

    Can be one of `default`, `force`, `none` or `ssl` (case
    insensitive). `default` is to try TLS (STARTTLS) but fall back to
    plain text depending on the SMTP server. `force` require TLS or
    will fail while `none` does not try to establish a secure session
    and may be rejected by modern mail servers. `ssl` uses implicit TLS
    (SMTPS) from the start of the connection, which is also used if
    `_smtp_port` is `465` and `_smtp_tls` is not `none`.

* `_smtp_port` - default depends on `_smtp_tls`

    If not set then the port is `587` for `default` and `force` (with
    `default` falling back to `25` if `587` cannot be reached), `465`
    for `ssl` and `25` for `none`. If set then only that port is used
    and a failure to connect is not retried on any other port.

* `_smtp_auth` - default is to select the strongest mechanism offered

    The SMTP authentication mechanism to use, one of `plain`, `login`,
    `cram-md5`, `xoauth2`, `scram-sha-1`, `scram-sha-256`, the `-plus`
    variants of the `scram` mechanisms, `ntlm`, or `auto`. The `plain`
    and `login` mechanisms are not used without TLS unless
    `plain-noenc` or `login-noenc` is given.

