# `geneos restore`

# `geneos restore` / `geneos load`

Restore one or more instances and/or shared directories from an archive created by `geneos backup`.

Use the `--list/-l` flag to show the high-level contents from the archive that match other arguments given.

Without either a component type or at least one instance name or pattern the command will do nothing. If given a component type without instance names all matching instances of that component type will be restored. Without a component type but with instances names, all matching instances, regardless of component type, will be restored. To restore all instances use the name `all`. Instance names can use wildcard patterns in shell (or "glob") format. Using the `--shared/-s` flag to restore any shared files and directories only applies to the matching component type if it given.

If an instance name is given in the format `DEST=SRC` then instances in the archive called `SRC` will be restored but renamed `DEST`. In this case `SRC` cannot be a wildcard.

Existing instances will not be overwritten, similarly for shared directories.

The command accepts a combination of filenames and instance name patterns, with optional renaming, and distinguishes them by validating the arguments. Any arguments that are not valid instance names (or wildcard or rename patterns) are treated as archive files. In case your archive file matches a valid instance name you should either use an absolute path to the file or a `./` prefix.

To read from STDIN use `-` but then you must also specify the compression type used with the `--decompress/-z` flag. Supported values are `gzip`, `bzip2` and `none`.

When restoring an instance any changes to the paths in the instance configuration will be updated to match the destination host's `geneos` root. No component specific files will be changed, including `gateway.setup.xml` and so on.

```text
geneos restore [flags] [TYPE] [[DEST=]NAME...]
```

### Options

```text
  -s, --shared            include shared files
  -z, --decompress TYPE   use decompression TYPE, one of `gzip`, `bzip2` or `none`
                          if not given then the file name is used to guess the type
                          MUST be supplied if the source is stdin (`-`)
  -l, --list              list the contents of the archive(s)
```

## Examples

```bash
geneos restore backup.tgz
geneos restore gateway ABC x.tgz

```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
