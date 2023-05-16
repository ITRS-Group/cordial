## geneos package ls

List packages available for update command

### Synopsis


List the packages for the matching TYPE or all component types if no
TYPE is given. The `-H` flags restricts the check to a specific
remote host.

All timestamps are displayed in UTC to avoid filesystem confusion
between local summer/winter times in some locales.

Versions are listed in descending order for each component type, i.e.
`latest` is always the first entry for each component.


```
geneos package ls [flags] [TYPE]
```

### Options

```
  -H, --host string   Apply only on remote host. "all" (the default) means all remote hosts and locally (default "all")
  -j, --json          Output JSON
  -i, --pretty        Output indented JSON
  -c, --csv           Output CSV
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos package](geneos_package.md)	 - A brief description of your command

