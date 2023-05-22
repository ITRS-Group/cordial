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

    If set then authentication is attempted. The authentication methods
    supported are those supported by the Go
    [`net/smtp`](https://pkg.go.dev/net/smtp#Auth) package.

* `_smtp_password` - no default

    The password is only used if `_smtp_username` is defined. It should
    be stored encrypted using the "expandable" format created using [`geneos aes
    password`](/tools/geneos/docs/geneos_aes_password.md) program

* `_smtp_tls` - default `default`

    Can be one of `default`, `force` or `none` (case insensitive).
    `default` is to try TLS but fall back to plain text depending on the
    SMTP server. `force` require TLS or will fail while `none` does not
    try to establish a secure session and may be rejected by modern mail
    servers.

