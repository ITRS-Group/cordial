# `geneos aes`

The `aes` sub-system allows you to manage AES256 keyfiles and perform encryption and decryption.

The `aes` commands provide tools to manage Geneos AES256 key files as documented in the [Secure Passwords](https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm) section of the manuals.

In addition to the functionality built-in to Geneos as described in the Gateway documentation these encoded password can also be included in configuration files so that plain text passwords and other credentials are not visible to users.


## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos aes decode`](geneos_aes_decode.md)	 | Decode a Geneos AES256 format password using a key file |
| [`geneos aes encode`](geneos_aes_encode.md)	 | Encode plaintext to a Geneos AES256 password using a key file |
| [`geneos aes import`](geneos_aes_import.md)	 | Import key files for component TYPE |
| [`geneos aes list / ls`](geneos_aes_list.md)	 | List key files |
| [`geneos aes new`](geneos_aes_new.md)	 | Create a new key file |
| [`geneos aes password / passwd`](geneos_aes_password.md)	 | Encode a password with an AES256 key file |
| [`geneos aes set`](geneos_aes_set.md)	 | Set active keyfile for instances |

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
