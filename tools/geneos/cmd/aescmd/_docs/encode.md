Encode plaintext to a Geneos AES256 format password using a key file, or create a Gateway "app key" file.

A key file should be provided using the `-keyfile`/`-k` option for a file path, the `--crc`/`-c` option for the CRC of a shared key file, or otherwise all matching instances that have a configured key file are used to produce an encrypted password.

Without matching `TYPE` or `NAME` the encode command with not update all instances. To force this, use `all` as an explicit wildcard.

## Encoding Passwords

For encoding passwords, the plaintext password can be provided in one three ways:

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

## App Keys

To create an app key file suitable for connecting to an SSO Agent, Gateway Hub or Obcerv from a Gateway for Centralised Configuration basic authentication support use the `--app-key`/`-A` flag. The value passed with the flag must be a valid provider, which is one of: "`ssoAgent`", "`gatewayHub`" or "`obcerv`". These values are case-sensitive.

The client ID and client secret can either be passed on the command line using the `--client-id`/`-C` and `--client-secret`/`-S` flags respectively, or you will be prompted to enter one or both using a non-echoing password-like dialogue.

The app key file contents are written to STDOUT unless you supply an `-app-key-file`/`-a` filename. This should be a file name and not a file path, and will be used to write an app key file in each matching instance home directory. If you supply a file path then the results are undetermined.

The contents of the app key output of saved file should be identical to that of the `-store-app-key` Gateway command line option but using an external key file (only).
