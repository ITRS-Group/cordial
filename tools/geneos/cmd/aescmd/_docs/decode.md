# `geneos aes decode`

Decode a Geneos AES256 format password using the key file(s) given.

If an `expandable` string is given with the `--expandable`/`-e` option it must be of the form `${enc:...}` (be careful to single-quote this string when using a shell) and is then decoded using the key file(s) listed and the ciphertext in the value. All other flags and arguments are ignored.

The format of `expandable` strings is documented here:

<https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#readme-expandable-formats>

A specific key file can be given using the `--keyfile`/`-k` flag and an alternative ("previous") key file with the `--previous`/`-v` flag. If either of these key files are supplied then the command tries to decode the given ciphertext and a value may be returned. An error is returned if all attempts fail.

Finally, if no key files are provided then matching instances are checked for configured key files and each one tried or the default key file paths are tried. An error is only returned if all attempts to decode fail. The ciphertext may contain the optional prefix `+encs+`. If both `-p` and `-s` options are given then the argument to the `--password`/`-p` flag is used. To read a ciphertext from STDIN use `--source -`/`-s -`.
