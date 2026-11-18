# `config` package

The config package extends `viper` with local customisations.

Unless the file format is given to [`Load`](https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#Load) then all the `viper` formats are supported.

## Expansion of values

The `Get`, `Lookup` and `Expand` functions add expansion of embedded strings of the form `${name}` (but not the bare form `$name`) to the values from the configuration. The string values, including those in lists and maps and in structures passed to `UnmarshalKey`, are expanded using [`Expand`](https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#Expand) which supports a range of data substitutions, described below. Use the `NoExpand()` option to retrieve the underlying value.

## Defaults and precedence

When reading a configuration with [`Read`](https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#Read) the order of precedence, highest first, of the settings in the result is:

1. values set explicitly using `Set`, and environment variables if enabled using `WithEnvs()`
2. the main configuration file, or the `Reader()`
3. defaults files, i.e. `NAME.defaults.EXT`
4. the embedded defaults given using `WithDefaults()`
5. the settings of the configuration passed to `DefaultsFrom()`

Using `UseDefaults(false)` means that none of the defaults (3, 4 and 5) are used. `Read` always returns a configuration, even if there is also an error.

`Write` writes all of the settings visible in a configuration, including any defaults and values from the environment. New files are created with permissions `0644` and directories with `0775` unless the `FileMode()` and `DirMode()` options are used, as they are for credentials files (`0600` and `0700`).

## Credentials

Credentials are supported with memguard Enclaves and LockedBuffers. There are
methods:

* [`GetPassword`](config.go#GetPassword)
* [`ExpandToEnclave`](expand.go#ExpandToEnclave)
* [`ExpandToLockedBuffer`](expand.go#ExpandToLockBuffer)

## Expandable Formats

The documentation below is reproduced from the source comments for
[`expand.go`](expand.go) and should be kept in-sync by package
maintainers (using `go doc pkg/config` as the primary source).

---

# Expandable Formats

The Expand family of functions return the input with all occurrences of the form
`${name}` replaced using an os.Expand-like function (but without support for `$name`
in the input) for the built-in and optional formats (in the order of priority)
below. The caller can use options to define additional expansion functions
based on a "prefix:", disable external lookups and also to pass in lookup tables
referred to as value maps.

* `${enc:keyfile[|keyfile...]:encodedvalue}`

    "encodedvalue" is an AES256 ciphertext in Geneos format - or, if
    not prefixed with `+encs+` then it is processed as an expandable
    string itself and can be a reference to another configuration key
    or a file or remote url containing one - which will be decoded
    using the key file(s) given. Each "keyfile" must be one of either
    an absolute path, a path relative to the working directory of the
    program, or if prefixed with "~/" then relative to the home
    directory of the user running the program. The first valid decode
    (see below) is returned.

    To minimise (but not wholly eliminate) any false-positive decodes
    that occur in some circumstances when using the wrong key file,
    the decoded value is only returned if it is a valid UTF-8 string
    as per [utf8.Valid](https://pkg.go.dev/unicode/utf8#Valid).

    This prefix can be disabled with the `config.NoDecode()` option.

    Note: The keyfile(s) can be in Windows file path format. The
    fields for keyfiles and the encoded value are split at the first
    colon (`:`) that is followed by `+encs+`, `config:`, `env:` or
    the prefix of any other enabled expansion (e.g. `file:` or
    `https:`), otherwise at the last colon in the string, and not any
    contained in a drive letter path. On Windows paths may also
    contain normal backslashes, they are not considered escape
    characters once extracted from the underlying configuration
    syntax (which may impose it's own rules, e.g. JSON)

    The decoded value is inserted exactly as it is and is never
    expanded further.

    Examples:

        password: ${enc:~/.keyfile:+encs+9F2C3871E105EC21E4F0D5A7921A937D}
        password: ${enc:/etc/geneos/keyfile.aes:env:ENCODED_PASSWORD}
        password: ${enc:~/.config/geneos/keyfile1.aes:app.password}
        password: ${enc:~/.keyfile.aes:config:mySecret}

* `${config:key} or ${path.to.config}`

    Fetch the "key" configuration value (for single layered
    configurations, where a sub-level dot cannot be used) or if any
    value containing one or more dots "." will be looked-up in the
    existing configuration that the method is called on. The underlying
    configuration is not changed and values are resolved each time
    Expand is called. No locking of the configuration is done. The
    value found is itself expanded, see "Nested expansion" below.

* `${key}`

    `key` will be substituted with the value of the first matching key
    from the tables set using `config.LookupTable()`, in the order passed
    to the function. The tables are checked first for any name, not
    only plain words, so a table can override a configuration key or a
    file path. If the name is not found in any table, or there are no
    tables, then a plain name (one with no `:` and no `/`, and not
    containing the key delimiter) is looked up as an environment
    variable, as below. The values from tables are inserted exactly as
    they are and are never expanded further.

* `${env:name}`

    `name` will be substituted with the contents of the environment
    variable of the same name. If no environment variable with name
    exists then the value returned is an empty string. The value is
    inserted exactly as it is and is never expanded further.

The additional prefixes below are enabled by default. They can be
disabled using the config.ExternalLookups() option.

* `${.../path/to/file}` or `${~/file}` or `${file://path/to/file}` or `${file:~/path/to/file}`

    The contents of the referenced file will be read. Multiline files
    are used as-is; this can, for example, be used to read PEM
    certificate files or keys. If the path is prefixed with `~/` (or
    as an addition to a standard file url, if the first `/` is
    replaced with a tilde `~`) then the path is relative to the home
    directory of the user running the process.

    Any name that contains a `/` but not a `:` will be treated as a
    file, if file reading is enabled. File paths can be absolute or
    relative to the working directory (or relative to the home
    directory, as above)

    For Windows file paths you must use the URL style `${file:...}`
    formats, otherwise any drive identifier will be treated as a
    partial expansion type and cause unexpected behaviour.

    Examples:

        certfile: ${file://etc/ssl/cert.pem}
        template: ${file:~/templates/autogen.gotmpl}
        relative: ${./file.txt}

* `${https://host/path}` or `${http://host/path}`

    The contents of the URL are fetched and used similarly as for local
    files above. The URL is passed to
    [http.Get](https://pkg.go.dev/net/http#Client.Get) and supports
    proxies, embedded Basic Authentication and other features from that
    function. As for files, the content is inserted exactly as it is
    and is never expanded further. The request times out after 30
    seconds.

The prefix below can be enabled with the config.Expressions() option.

* `${expr:EXPRESSION}`

    EXPRESSION is evaluated using <https://github.com/maja42/goval>. Inside
    the expression all configuration items are available as variables
    with the top level map `env` set to the environment variables
    available. All results are returned as strings. An empty string
    may mean there was an error in evaluating the expression.

Additional custom prefixes can be added with the config.Prefix()
option. A prefix, and any use of config.ExternalLookups() or
config.Expressions(), only applies to the Config or call it is used
with and is never seen by any other.

The bare form `$name` is NOT supported, unlike os.Expand as this can
unexpectedly match values containing valid literal dollar signs.

### Nested expansion

Expansion is nested, not just a single pass. Configuration values are
stored as literals and are expanded each time they are used and a value
that is itself found to contain `${...}` references is expanded in turn,
so, for example, `${config:home}/${config:logfile}` can be used in a value
that is referred to by another. The same applies to the results of
functions registered with `config.Prefix()`. The nesting is limited by the
`config.Loop()` option, which has a default of 10, counting the string
passed to Expand as the first level. At the limit the value found is
used as it is, without any further expansion, and so a configuration
item that refers to itself results in a reference left in the output
rather than an infinite loop. A limit of less than 1 means there is no
nested expansion.

Only the values from configuration items and from `config.Prefix()`
functions are expanded in this way. The values from lookup tables
(`config.LookupTable()`), the environment, files, URLs, expressions,
decoded `enc:` values and from functions registered with
`config.PrefixVerbatim()` are inserted exactly as they are. This means
that data from an untrusted source, such as from a network request used
in a lookup table, can never cause further expansion, e.g. of
`${file:...}` or `${config:password}`.

For each substitution any leading and trailing whitespace are removed,
unless `config.TrimSpace(false)` is used. External sources are fetched
each time they are used and so there may be a performance impact as well
as the value unexpectedly changing during a process lifetime.

The default value for a Get, from `config.DefaultValue()`, is never
expanded. The initial value, from `config.InitialValue()`, is.

### Errors and escaping

Any errors (particularly from substitutions from external files or
remote URLs) will result in an empty string being substituted, as does a
name that cannot be resolved, such as an unset environment variable or a
prefix that has not been registered. Error returns are intentionally
discarded. Where a value contains multiple expandable items processing
will continue even after an error for one of them. An opening `${` with
no closing brace is left as it is.

To have a literal `${` in the result use `$${`, which is replaced by `${`
with the remainder of the text left alone, e.g. `$${name}` is expanded to
`${name}`. As values are expanded as they are used, the escape is only
replaced once, at the level of the string that contains it, and the
result is never expanded again. For example, for YAML

    config:
      real: ${config.literal}
      literal: "$${unchanged}"

a reference to `${config.real}` will return the literal string
`${unchanged}`.

Inside the braces, any character, including a closing brace, can be
escaped using a backslash and nested pairs of braces are matched, so
`${replace:name:/(a)b/${1}c/}` is a single reference. The name, including
any escapes, is passed as-is to the function for the prefix and is not
expanded first.

