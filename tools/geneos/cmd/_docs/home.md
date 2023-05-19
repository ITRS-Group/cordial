Output a directory path for use in shell expansion like `cd $(geneos
home mygateway)`.

Without arguments, the output will be the root of the Geneos
installation, if defined, or an empty string if not. In the latter
case a shell running `cd` would interpret this as go to your home
directory.

With only a TYPE and no instance NAME the output is the directory
root directory of that TYPE, e.g. `${GENEOS_HOME}/gateway`

Otherwise, if the first NAME argument results in a match to an instance
then the output is it's working directory. If no instance matches the
first NAME argument then the Geneos root directory is output as if no
other options were given.

For obvious reasons this only applies to the local host and the
`--host`/`-H` option is ignored. If NAME is given with a host
qualifier and this is not `localhost` then this is treated as a
failure and the Geneos home directory is returned.

If the resulting path contains whitespace your shell will see this as
multiple arguments and a typical `cd` will fail. To avoid this wrap
the expansion in double quotes, e.g. `cd "$(geneos home 'Demo
Gateway')"`. The best solution is to not use white space in any
instance name or directory path above it. (Note: We tried outputting
a quoted path but the bash shell ignores these quotes inside
`$(...)`)
