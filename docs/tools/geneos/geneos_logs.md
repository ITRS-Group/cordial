## geneos logs

Show log(s) for instances

### Synopsis


Show log(s) for instances. The default is to show the last 10 lines
for each matching instance. If either `-g` or `-v` are given without
`-f` to follow live logs, then `-c` to search the whole log is
implied.
	
When more than one instance matches each output block is prefixed by
instance details.


```
geneos logs [flags] [TYPE] [NAME...]
```

### Options

```
  -n, --lines int       Lines to tail (default 10)
  -f, --follow          Follow file
  -c, --cat             Cat whole file
  -g, --match string    Match lines with STRING
  -v, --ignore string   Match lines without STRING
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

