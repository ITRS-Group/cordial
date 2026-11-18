# `gdna pubkey`

# `gdna pubkey`

Output the public key corresponding to a licd private key, if set in `gdna.licd-private-key` or, if that is empty, in the default key file `gdna-private-key.pem` in the `geneos` directory in your user configuration directory. This is for use with the licd from version 7.8.0 and later, which supports JWT authentication using public/private key pairs.

## Usage

```text
gdna pubkey
```

### Options

```text
  -f, --config FILE     Use configuration FILE
  -l, --logfile file    Write logs to file. Use '-' for console or /dev/null for none (default "gdna.log")
  -o, --output string   Output file for public key (default: stdout)
```

## SEE ALSO

* [gdna](gdna.md)	 - Process Geneos License Usage Data
