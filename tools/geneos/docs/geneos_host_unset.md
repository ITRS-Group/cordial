# `geneos host unset`

The `geneos host unset` command allows you to remove parameters from host configurations. This can be used to remove items like encrypted passwords as well as private key file paths. Like the main `geneos unset` command parameters have to be named using the `--key/-k` command line flag and to remove private key files from the list use `--privatekey/-i PATH`. At this time the paths to private key files must be given exactly as in the configuration and you cannot use wildcards.

```text
geneos host unset [flags] [TYPE] [NAME...]
```

### Options

```text
  -k, --key KEY           Unset configuration parameter KEY
                          (Repeat as required)
  -i, --privatekey PATH   Private key file
```

## Examples

```bash
geneos host unset rem2 -i /path/to/id_rsa

```

## SEE ALSO

* [geneos host](geneos_host.md)	 - Remote Host Operations
