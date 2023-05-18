# `geneos start`

Start instances

```text
geneos start [flags] [TYPE] [NAME...]
```

## Details

Start one or more matching instances. All instances are run in
the background and STDOUT and STDERR are redirected to a `.txt` file
in the instance directory. You can watch the resulting logs files with the
`-l` flag.

### Options

```text
  -l, --log   Run 'logs -f' after starting instance(s)
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
