# `geneos backup`

# `geneos backup` / `geneos save`

Backup instance data in an archive file, either for backup or to be restored on another system using the `geneos restore` command.

Each matching instance has it's primary configuration files included in the archive. By default no AES files, certificates or private keys are included in the archive (taken from the instance configuration and not by filename pattern). If saving any of these files is enabled using the flags below then only files in the instance directory are included; Files outside the instance directory are never saved. Files that match the clean or purge lists for that type are ignored unless the `--all/-a` flag is given in which case those files are not skipped (but sensitive files like AES key files and private keys are still only included using the specific options below).

The default output format is as a `tar.gz` file (gzipped tar). You can also select bzip2 or none using the `--compress/-c` option. Unless you set your own archive file name using the `--output/-o` flag the file name with be automatically generated based on how many unique component types and instance names there are. The `--output/-o` flags will also accept `-` to mean output to stdout.

If a single instance is selected than the default destination archive is made up of `geneos`, the component type and the instance name, in the format `geneos-TYPE-NAME.tar.gz` in the current directory. Use the `--output`/`-o` option to override, including a `-` to indicate STDOUT for piping to another command. In this case any other messages are written to STDERR. If written to a file. any existing file with the same name is overwritten. If the program is invoked using another name then that is used instead of `geneos`. All spaces in instance names are replaced with underscores. So, a Gateway with the name "Demo Gateway" would be saved as `geneos-gateway-Demo_Gateway.tar.gz`.

You can add a time and date to the file name using the `--datetime/-D` flag, and this is in the fixed numeric format like `YYYYMMDDhhmmss`.

If there are no instances selected then the archive is named either simple `geneos.tar.gz` if multiple components match or `geneos-TYPEs.tar.gz` (note the plural of `TYPEs`, e.g. `gateways`) if the instances are all of the same type.

Instances matching across multiple hosts are not supported, and this returns an error. If the instance (or all matching instances) are on the same remote host then these are all saved and the default filename will include the host label (the `geneos host` name).

To include the component shared directory/directories, use the `--shared` option.

To include AES files, use the `--aes` option. Without this option the files that are skipped are those referenced by the instance `keyfile` and `prevkeyfile` parameters. When using `--shared` files with an `.aes` extension or any `keyfile` directory and it's contents are skipped.

To include certificate and private key files, use the `--tls` option. Without this option the files skipped are those referenced by the instance parameters `certificate`, `privatekey` and `certchain`. When using `--shared` all files with the extensions `.pem`, `.key` and `.crt` are skipped.

File over 2MiB are not included in the archive unless the `--all/-a` option is used, but this limit can be controlled using the `--size/-s` option which accepts all the common units, e.g. `50KB` or `10MiB`. Using a size of `0` (zero) removes the limit.

The contents of the archive are relative to the root of the Geneos installation, and the `geneos load` command will refactor any changes to paths in the instance configuration JSON files.

```text
geneos backup [flags] [all] | [TYPE] [NAME...]
```

### Options

```text
  -o, --output string   Write to "DEST". Without a destination filename the command creates
                        a file name based on the contents of the archive. If DEST is a directory
                        or has a "/" suffix then the file is written to that directory using the
                        same naming format as if no file was given. Directories are created
                        as required.
  -D, --datetime        Add a datetime string (YYYYMMDDhhmmss) the auto-generated file names
  -z, --compress type   Compression type. One of `gzip`, `bzip2` or `none`. (default "gzip")
  -s, --size string     Skip files larger than this size unless --all is used. Accepts suffixes
                        with common scale units such as K, M, G with both B and iB units,
                        e.g. "2MiB". "0" (zero) means no limit to file sizes. (default "2MiB")
  -A, --all             Include all files except AES key files, certificates and private keys.
      --shared          Include per-component shared directories from outside instance directories.
                        
      --aes             Include AES key files.
      --tls             Include certificates, private keys and certificate chains.
```

## Examples

```bash
geneos backup gateway gw1
geneos backup all

```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
