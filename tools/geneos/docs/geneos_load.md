# `geneos load`

Load one or more instances from an archive created by `save`.

```text
geneos load [flags] [TYPE] [[DEST=]NAME...]
```

### Options

```text
  -i, --input FILE        import one or more instances from FILE
                          FILE can be `-` for STDIN
  -s, --shared            include shared files when using --instances
  -z, --decompress type   use decompression type, one of `gzip`, `bzip2` or `none`
                          if not given then the file name is used to guess the type
```

## Examples

```bash

geneos load gateway ABC x.tgz


```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
