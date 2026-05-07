# `geneos aes password`

The `aes password` command encodes a password using the key file in the user's config directory, which by default is `${HOME}/.config/geneos`. If no key file exists it is created. Output is in "expandable" format.

You will be prompted to enter the password (twice, for validation) unless one of the flags is set to select an alternative source for the plaintext.

To encode a plaintext password using a specific key file please use the `geneos aes encode` command

## Usage

```text
geneos aes password [flags]
```

### Options

```text
      --allow-root          allow running as root (not recommended)
  -G, --config string       config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME       Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
  -p, --password SECRET     Password
  -s, --source PATH|URL|-   External source for password PATH|URL|-
```

## SEE ALSO

* [geneos aes](geneos_aes.md)	 - AES256 Key File Operations
