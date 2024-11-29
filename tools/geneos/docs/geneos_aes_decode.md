# `geneos aes decode`

Decode a Geneos AES256 format password using the key file(s) given.

If an `expandable` string is given with the `--expandable`/`-e` option it must be of the form `${enc:...}` (be careful to single-quote this string when using a shell) and is then decoded using the key file(s) listed and the ciphertext in the value. All other flags and arguments are ignored.

The format of `expandable` strings is documented here:

<https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#readme-expandable-formats>

A specific key file can be given using the `--keyfile`/`-k` flag and an alternative ("previous") key file with the `--previous`/`-v` flag. If either of these key files are supplied then the command tries to decode the given ciphertext and a value may be returned. An error is returned if all attempts fail.

Finally, if no key files are provided then matching instances are checked for configured key files and each one tried or the default key file paths are tried. An error is only returned if all attempts to decode fail. The ciphertext may contain the optional prefix `+encs+`. If both `-p` and `-s` options are given then the argument to the `--password`/`-p` flag is used. To read a ciphertext from STDIN use `--source -`/`-s -`.

```text
geneos aes decode [flags] [TYPE] [NAME...]
```

### Options

```text
  -e, --expandable string   The keyfile and ciphertext in expandable format (including '${...}')
  -k, --keyfile KEYFILE     Path to keyfile (default /home/peter/.config/docs/keyfile.aes)
  -v, --previous KEYFILE    Path to previous keyfile (default /home/peter/.config/docs/prevkeyfile.aes)
  -p, --password string     Geneos formatted AES256 password
  -s, --source string       Alternative source for password
```

## Examples

```bash
# don't forget to use single quotes to escape the ${...} from shell
# interpolation
geneos aes decode -e '${enc:~/.config/geneos/keyfile.aes:hexencodedciphertext}'

# decode from the environment variable "MY_ENCODED_PASSWORD"
geneos aes decode -e '${enc:~/.config/geneos/keyfile.aes:env:MY_ENCODED_PASSWORD}'

# try to decode using AES key file configured for all instances
geneos aes decode -p +encs+hexencodedciphertext

# try to decode using the AES key file associated with the 'Demo Gateway' instance
geneos aes decode gateway 'Demo Gateway' -p +encs+hexencodedciphertext

```

## SEE ALSO

* [geneos aes](geneos_aes.md)	 - AES256 Key File Operations
