Encode plaintext to a Geneos AES256 format password using a key file.

A key file must either be provided using the `-k` option or otherwise all matching instances that have a configured key file are used to produce an encrypted password.

The plaintext password can be provided in three ways.

1. The default is to prompt for the plaintext and again to verify they match.
2. Alternatively the password can be provided directly on the command line using the `-p plaintext` flag or,
3. From an external source using the `-s PATH` or `-s URL` option where the contents of the file at PATH ir URL is read and used. If `-s -` is used then the plaintext is read from `STDIN`.

It is important to note that no whitespace is trimmed from the plaintext. This can have unexpected results if you do something like this:

```bash
$ echo "test" | geneos aes encode -s -
```

Rather than:

```bash
$ echo -n "test" | geneos aes encode -s -
```