The `aes password` command encodes a password using the key file in the user's config directory, which by default is `${HOME}/.config/geneos`. If no key file exists it is created. Output is in "expandable" format.

You will be prompted to enter the password (twice, for validation) unless one of the flags is set to select an alternative source for the plaintext.

To encode a plaintext password using a specific key file please use the `geneos aes encode` command
