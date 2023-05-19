# `geneos aes password`

Encode a password with an AES256 key file

```text
geneos aes password [flags]
```

Encode a password using the user's keyfile. If no keyfile exists it is
created. Output is in `expandable` format.

You will be prompted to enter the password (twice, for validation)
unless on of the flags is set.

To encode a plaintext password using a specific key file please use the
`geneos aes encode` command

### Options

```text
  -p, --password PLAINTEXT   A plaintext password
  -s, --source PATH|URL|-    External source for plaintext PATH|URL|-
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos aes](geneos_aes.md)	 - Manage Geneos compatible key files and encode/decode passwords
