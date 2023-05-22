# `config` package

The config package extends `viper` with local customisations.

Unless the file format is given to [`Load`](https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#Load) then all the `viper` formats are supported.

## Expansion of values

The following package functions and type methods are overridden to add expansion of embedded strings of the form `${name}` and `$name`:

* [`GetString`](config.go#GetString)
* [`GetStringSlice`](config.go#GetStringSlice)
* [`GetStringMapString`](config.go#GetStringMapString)

The string values are passed to [`ExpandString`](https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#Config.ExpandString) which supports a range of data substitutions.

## Credentials

Credentials are supported with memguard Enclaves and LockedBuffers. There are
methods:

* [`GetPassword`](config.go#GetPassword)
* [`ExpandToEnclave`](expand.go#ExpandToEnclave)
* [`ExpandToLockedBuffer`](expand.go#ExpandToLockBuffer)

## Expandable Formats

This documentation is reproduced from the source comments for
[`expand.go`](expand.go) and should be kept in-sync by
package maintainers.

---

The Expand* functions returns the input with all occurrences of the form
`${name}` replaced using an
[os.Expand](https://pkg.go.dev/os#Expand)-like function (but without
support for $name in the input) for the built-in and optional formats
(in the order of priority) below. The caller can use options to define
additional expansion functions based on a `prefix:`, disable external
lookups and also to pass in lookup tables referred to as value maps.

* `${enc:keyfile[|keyfile...]:encodedvalue}`

    `encodedvalue` is an AES256 ciphertext in Geneos format - or, if
    not prefixed with `+encs+` then it is processed as an expandable
    string itself and can be a reference to another configuration key
    or a file or remote url containing one - which will be decoded
    using the key file(s) given. Each `keyfile` must be one of either
    an absolute path, a path relative to the working directory of the
    program, or if prefixed with `~/` then relative to the home
    directory of the user running the program. The first valid decode
    (see below) is returned.

    To minimise (but not wholly eliminate) any false-positive decodes
    that occur in some circumstances when using the wrong key file,
    the decoded value is only returned if it is a valid UTF-8 string
    as per [utf8.Valid](https://pkg.go.dev/unicode/utf8#Valid).

    Examples:

    ```yaml
    password: ${enc:~/.keyfile:+encs+9F2C3871E105EC21E4F0D5A7921A937D}
    password: ${enc:/etc/geneos/keyfile.aes:env:ENCODED_PASSWORD}
    password: ${enc:~/.config/geneos/keyfile1.aes:app.password}
    password: ${enc:~/.keyfile.aes:config:mySecret}
    ```

    This prefix can be disabled with the `config.NoDecode()` option.

* `${config:key} or ${path.to.config}`

    Fetch the `key` configuration value (for single layered
    configurations, where a sub-level dot cannot be used) or if any
    value containing one or more dots `.` will be looked-up in the
    existing configuration that the method is called on. The
    underlying configuration is not changed and values are resolved
    each time ExpandString() is called. No locking of the
    configuration is done.

* `${key}`

    `key` will be substituted with the value of the first matching key
    from the tables set using the `config.LookupTable()` option, in the
    order passed to the function. If no lookup tables are set (as
    opposed to the key not being found in any of the tables) then name
    is looked up as an environment variable, as below.

* ${env:name}

    `name` will be substituted with the contents of the environment
    variable of the same name. If no environment variable with name
    exists then the value returned is an empty string.

The additional prefixes below are enabled by default. They can be
disabled using the config.ExternalLookups() option.

* `${.../path/to/file} or ${~/file} or ${file://path/to/file} or ${file:~/path/to/file}`

    The contents of the referenced file will be read. Multiline files
    are used as-is; this can, for example, be used to read PEM
    certificate files or keys. If the path is prefixed with "~/" (or
    as an addition to a standard file url, if the first "/" is
    replaced with a tilde "~") then the path is relative to the home
    directory of the user running the process.

    Any name that contains a `/` but not a `:` will be treated as a
    file, if file reading is enabled. File paths can be absolute or
    relative to the working directory (or relative to the home
    directory, as above)

    Examples:

    ```yaml
    certfile: ${file://etc/ssl/cert.pem}
    template: ${file:~/templates/autogen.gotmpl}
    relative: ${./file.txt}
    ```

* `${https://host/path} or ${http://host/path}`

    The contents of the URL are fetched and used similarly as for local
    files above. The URL is passed to
    [http.Get](https://pkg.go.dev/net/http#Get) and supports proxies,
    embedded Basic Authentication and other features from that function.

The prefix below can be enabled with the `config.Expressions()` option.

* `${expr:EXPRESSION}`

    `EXPRESSION` is evaluated using
    <https://pkg.go.dev/github.com/maja42/goval>. Inside the expression
    all configuration items are available as variables with the top
    level map `env` set to the environment variables available. All
    results are returned as strings. An empty string may mean there was
    an error in evaluating the expression.

Additional custom prefixes can be added with the `config.Prefix()`
option.

The bare form `$name` is NOT supported, unlike
[os.Expand](https://pkg.go.dev/os#Expand) as this can unexpectedly match
values containing valid literal dollar signs.

Expansion is not recursive. Configuration values are read and stored
as literals and are expanded each time they are used. For each
substitution any leading and trailing whitespace are removed.
External sources are fetched each time they are used and so there may
be a performance impact as well as the value unexpectedly changing
during a process lifetime.

Any errors (particularly from substitutions from external files or
remote URLs) will result in an empty or corrupt string being returned.
Error returns are intentionally discarded and an empty string
substituted. Where a value contains multiple expandable items processing
will continue even after an error for one of them.

It is not currently possible to escape the syntax supported by
ExpandString and if it is necessary to have a configuration value be
a literal of the form `${name}` then you can set an otherwise unused
item to the value and refer to it using the dotted syntax, e.g. for
YAML

```yaml
config:
    real: ${config.literal}
    literal: "${unchanged}"
```

In the above a reference to `${config.real}` will return the literal
string `${unchanged}` as there is no recursive lookups.
