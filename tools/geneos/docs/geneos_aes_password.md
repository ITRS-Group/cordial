# `geneos aes password`

Encode a password using the user's keyfile. If no keyfile exists it is created. Output is in `expandable` format.

You will be prompted to enter the password (twice, for validation) unless on of the flags is set.

To encode a plaintext password using a specific key file please use the `geneos aes encode` command

```text
geneos aes password [flags]
```

### Options

```text
  -p, --password PLAINTEXT   A plaintext password
  -s, --source PATH|URL|-    External source for plaintext PATH|URL|-
```

## SEE ALSO

* [geneos aes](geneos_aes.md)	 - AES256 Key File Operations
