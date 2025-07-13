# `geneos save`

Save instance data in an archive file, either for backups or to be loaded on another system using the `geneos load` command.

By default no AES files, certificates or private keys are included in the archive (taken from the instance configuration and not by filename pattern). If saving any of these files is enabled using the flags below then only files in the instance directory are included; Files outside the instance directory are never saved.

Files in the that match the clean or purge lists for that type are ignored.

The default output format is gzipped tar. You can also select bzip2 or none using the `--compress/-c` option.

If a single instance is selected than the default destination archive is made up of `geneos`, the component type and the instance name, in the format `geneos-TYPE-NAME.tar.gz` in the current directory. Use the `--output`/`-o` option to override, including a `-` to indicate STDOUT for piping to another command. In this case any other messages are written to STDERR. If written to a file. any existing file with the same name is overwritten. If the program is invoked using another name then that is used instead of `geneos`. All spaces in instance names are replaced with underscores. So, a Gateway with the name "Demo Gateway" would be saved as `geneos-gateway-Demo_Gateway.tar.gz`

If there are no instances selected then the archive is named either simple `geneos.tar.gz` if multiple components match or `geneos-TYPEs.tar.gz` (note the plural of `TYPEs`, e.g. `gateways`) if the instances are all of the same type.

Instances matching across multiple hosts are not supported, and this returns an error. If the instance (or all matching instances) are on the same remote host then these are all saved and no indication of the source host is recorded in the file or the file name.

To include the component shared directory/directories, use the `--shared` option.

To include AES files, use the `--aes` option.

To include certificate and private key files, use the `--tls` option.

The contents of the archive are relative to the root of the Geneos installation, and the `geneos load` command will refactor any changes to paths in the instance configuration JSON files.

```bash
geneos save gateway GATEWAY1
```
