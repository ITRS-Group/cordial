# `geneos save`

Export one or more instances as archives, either for backups or - in a future release - to be imported on another system using the `geneos import --instance` command.

By default no aes files, certificates or private keys are included in the archive (taken from the instance configuration and not by filename pattern) and also, optionally, any files found in the component shared directory will be included. Additionally, any files in the instance directory that match the clean or purge lists for that type are ignored.

The output is in tar format, compressed using gzip bye default. You can also select bzip2 or none using the `--compress/-c` option. 

If a single instance is selected than the default destination archive is made up of `geneos`, the component type and the instance name, in the format `geneos-TYPE-NAME.tar.gz` in the current directory. Use the `--output`/`-o` option to override, including a `-` to indicate STDOUT for piping to another command. In this case any other messages are written to STDERR. If written to a file. any existing file with the same name is overwritten. If the program is invoked using another name then that is used instead of `geneos`. All spaces in instance names are replaced with underscores. So, a Gateway with the name "Demo Gateway" would be exported as `geneos-gateway-Demo_Gateway.tar.gz`

If there are instances selected then the archive is named either simple `geneos.tar.gz` if multiple components match or `geneos-TYPEs.tar.gz` (note the plural of `TYPEs`, e.g. `gateways`) if the instances are all of the same type.

Instances matching across multiple hosts are not supported, and this returns an error. If the instance (or all matching instances) are on the same remote host then these are all exported and no indication of the source host is recorded in the file or the file name.

To include the component shared directory/directories, use the `--shared` option.

To include AES files, use the `--aes` option.

To include certificate and private key files, use the `--tls` option.

The contents of the archive are relative to the root of the Geneos installation.

```bash
geneos export gateway GATEWAY1
```

```text
geneos save [flags] [TYPE] [NAME...]
```

### Options

```text
  -o, --output path     Output file path. If path is a directory or has a '/' suffix then the constructed
                        file name is used in that directory. If not final file name is given then the
                        file name is of the form 'geneos[-TYPE][-NAME].tar.gz'
  -D, --datetime        include a datetime string the in the auto-generated archive name
  -z, --compress type   Compression type. One of `gzip`, `bzip2` or `none`. (default "gzip")
  -s, --size string     Ignore files larger than this size (in bytes) unless --all is used
                        Accepts suffixes i=with both B and iB units (default "1MiB")
  -A, --all             Include all files except AES key files, certificates and associated files, in the archive.
                        This may fail for running instances
      --shared          Include shared directory contents in the archive
                        (also use --all to include files that are filtered by clean/purge lists)
      --aes             Include AES key files in the archive
                        (never includes user's own keyfile)
      --tls             Include certificates, private keys and certificate chains in archive
```

## Examples

```bash

```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
