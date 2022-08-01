# config package

The config package wraps the common viper methods and types but adds some custom processing.

This probably could be done using custom decode hooks for viper, but I have not had time to understand them in detail.

Using a wrapper also allows custom methods on config for further functionality.

Unless the file format is given to LoadConfig() then all the viper formats are supported.

## Expansion of values

The methods below are overridden to add extra expansion of the values given:

* `GetString()`
* `GetStringSlice()`
* `GetStringMapString()`

The string values are passed to `ExpandString()` which supports the following embedded formats (which can appear mixed in a single value):

* `text` - text is passed back unchanged
* `${path.to.name}` - any value containing a dot (`.`) is used as a lookup to another configuration item. This is not recursive and if the value referenced is not a plain string it is returned as-is no interpolation is done on that value. This is primarily to avoid infinite loops. The value has leading and trailing spaces trimmed.
* `${env:ENV}` or `${name}` - the contents of the environment variable `ENV` (or `name`) are substituted - but see below for the treatment of the bare word form, without the `env:` prefix. The value has leading and trailing spaces trimmed.
* `${file:path/to/file}` or `${file:~/path/to/file}` - the contents of the given file are substituted after having leading and trailing spaces trimmed. If the path starts `~/` then the path is relative to the process owners home directory.
* `${http://path}` or `${https://path}` - the contents of the remote URL are fetched and substituted as for files above. If the URL includes standard `username@password` then these may/should be used as basic authentication as per Go [net/url](https://pkg.go.dev/net.url) and [net/http](https://pkg.go.dev/net/http) packages.

The replacement methods also take an optional argument called a confmap (configuration map) which is a `map[string]string` and this when expanding a value containing `${name}` and confmap is not empty then these values are used and environment variables are not interpolated.
