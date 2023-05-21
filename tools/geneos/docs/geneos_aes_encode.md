# `geneos aes encode`

Encode plaintext to a Geneos AES256 password using a key file

```text
geneos aes encode [flags] [TYPE] [NAME...]
```

Encode plaintext to a Geneos AES256 format password using a key file.

A key file must either be provided using the `-k` option or otherwise
all matching instances that have a configured key file are used to
produce an encrypted password.

The plaintext password can be provided in three ways. The default is to
prompt for the plaintext and again to verify they match. Alternatively
the password can be provided directly on the command line using the `-p
plaintext` flag or from an external source using the `-s PATH` or `-s
URL` option where the contents of the file at PATH ir URL is read and
used. If `-s -` is used then the plaintext is read from `STDIN`.

It is important to note that no whitespace is trimmed from the
plaintext. This can have unexpected results if you do something like
this:

$ echo "test" ` geneos aes encode -s -

rather then this:

$ echo -n "test" ` geneos aes encode -s -

### Options

```text
  -e, --expandable           Output in 'expandable' format
  -k, --keyfile KEYFILE      Path to keyfile
  -p, --password PLAINTEXT   Plaintext password
  -s, --source string        Alternative source for plaintext password
  -o, --once                 Only prompt for password once, do not verify. Normally use '-s -' for stdin
```

## Examples

```bash

```

## SEE ALSO

* [geneos aes](geneos_aes.md)	 - Manage Geneos compatible key files and encode/decode passwords
