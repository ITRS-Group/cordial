## geneos aes ls

List configured keyfiles for instances

### Synopsis


For matching instances list configured keyfiles, their location in
the filesystem and their CRC. 

The default output is human readable columns but can be in CSV
format using the `-c` flag or JSON with the `-j` or `-i` flags, the
latter "pretty" formatting the output over multiple, indented lines


```
geneos aes ls [flags] [TYPE] [NAME...]
```

### Options

```
  -c, --csv      Output CSV
  -j, --json     Output JSON
  -i, --pretty   Output indented JSON
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos aes](geneos_aes.md)	 - Manage Geneos compatible key files and encode/decode passwords

