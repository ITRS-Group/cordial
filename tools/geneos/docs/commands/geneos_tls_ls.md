## geneos tls ls

List certificates

### Synopsis


List certificates and their details. The root and signing
certs are only shown if the `-a` flag is given. A list with more
details can be seen with the `-l` flag, otherwise options are the
same as for the main ls command.


```
geneos tls ls [flags] [TYPE] [NAME...]
```

### Options

```
  -a, --all      Show all certs, including global and signing certs
  -l, --long     Long output
  -j, --json     Output JSON
  -i, --pretty   Output indented JSON
  -c, --csv      Output CSV
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

### SEE ALSO

* [geneos tls](geneos_tls.md)	 - Manage certificates for secure connections

