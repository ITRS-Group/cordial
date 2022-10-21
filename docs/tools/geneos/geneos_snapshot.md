## geneos snapshot

Capture a snapshot of each matching dataview

### Synopsis


Using the Dataview Snapshot REST endpoint in GA5.14+ Gateways,
capture each dataview matching to given XPATH(s). Options to select
what data to request and authentication.

Authentication details are taken from the instance configuration
`snapshot.username` and `snapshot.password` parameters. If either is
unset then they are taken from the command line or the user or global
configuration parameters of the same names - in that order.


```
geneos snapshot [flags] [gateway] [NAME] XPATH...
```

### Options

```
  -V, --value             Request cell values (default true)
  -S, --severity          Request cell severities
  -Z, --snooze            Request cell snooze info
  -U, --userassignment    Request cell user assignment info
  -u, --username string   Username for snaptshot, defaults to configuration value in snapshot.username
  -P, --pwfile string     Password file to read for snapshots, defaults to configuration value in snapshot.password or otherwise prompts
  -l, --limit int         limit matching items to display. default is unlimited. results unsorted.
  -x, --xpaths            just show matching xpaths
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

