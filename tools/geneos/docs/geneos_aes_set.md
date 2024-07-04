# `geneos aes set`

Import a key file and set parameters on matching instances.

To create a key file with new contents use the `aes new` command.

If the `--shared`/`-s` flag is set then the provided key file is imported to matching component shared key file directories and matching instances have their key file parameters set to point to the new location.

If `--crc CRC`/`-c CRC` is given then the 8 hex-digit CRC is used to look up an existing key file in the component's shared key file directory and if found then the matching instances are updated to use this. In this case no key files are changed.

Depending on the `--no-roll`/`-N` flag, any matching instances may have their `prevkeyfile` parameter updated to reference any original key file and, if a new key file is written in a non-shared location the previous file is also renamed using the suffix in the `--backup`/`-b` flag.

Key files are only set on components that support them.

```text
geneos aes set [flags] [TYPE] [NAME...]
```

### Options

```text
  -k, --keyfile PATH|URL|-   Key file to use. PATH|URL|-
                             Path to a local file, a URL or a dash for STDIN. (default "-")
  -c, --crc CRC              CRC of an existing shared keyfile to use
  -b, --backup string        Backup any existing keyfile with extension given (default "-prev")
  -N, --no-roll              Do not roll any existing keyfile to previous keyfile setting
  -s, --shared               Set as a shared keyfile, using the CRC as the file name prefix
```

## SEE ALSO

* [geneos aes](geneos_aes.md)	 - AES256 Key File Operations
