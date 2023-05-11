# config package

The config package wraps the common viper methods and types but adds some custom processing.

This probably could be done using custom decode hooks for viper, but I have not had time to understand them in detail.

Using a wrapper also allows custom methods on config for further functionality.

Unless the file format is given to [`Load`](https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#Load) then all the viper formats are supported.

## Expansion of values

The following package functions and type methods are overridden to add expansion of embedded strings of the form `${name}` and `$name`:

* [`GetString`](https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#GetString)
* [`GetStringSlice`](https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#GetStringSlice)
* [`GetStringMapString`](https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#GetStringMapString)

The string values are passed to [`ExpandString`](https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#Config.ExpandString) which supports a range of data substitutions.
