## geneos start

Start instances

### Synopsis


Start one or more matching instances. All instances are run in
the background and STDOUT and STDERR are redirected to a `.txt` file
in the instance directory. You can watch the resulting logs files with the
`-l` flag.


```
geneos start [flags] [TYPE] [NAME...]
```

### Options

```
  -l, --log    Run 'logs -f' after starting instance(s)
  -h, --help   help for start
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

