## geneos home

Print the home directory of the first instance or the Geneos home dir

### Synopsis

Output the home directory of the first matching instance or local
installation or the remote on stdout. This is intended for scripting,
e.g.

	cd $(geneos home)
	cd $(geneos home gateway example1)
		
No errors are logged. An error, for example no matching instance found, result in the Geneos
root directory being printed.

```
geneos home [TYPE] [NAME]
```

### Options

```
  -h, --help   help for home
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

