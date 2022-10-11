## geneos import

Import file(s) to an instance or a common directory

### Synopsis

Import file(s) to the instance or common directory. This can be used
to add configuration or license files or scripts for gateways and
netprobes to run. The SOURCE can be a local path or a url or a '-'
for stdin. DEST is local pathname ending in either a filename or a
directory. Is the SRC is '-' then a DEST must be provided. If DEST
includes a path then it must be relative and cannot contain '..'.
Examples:

	geneos import gateway example1 https://example.com/myfiles/gateway.setup.xml
	geneos import licd example2 geneos.lic=license.txt
	geneos import netprobe example3 scripts/=myscript.sh
	geneos import san localhost ./netprobe.setup.xml
	geneos import gateway -c shared common_include.xml

To distinguish SOURCE from an instance name a bare filename in the
current directory MUST be prefixed with './'. A file in a directory
(relative or absolute) or a URL are seen as invalid instance names
and become paths automatically. Directories are created as required.
If run as root, directories and files ownership is set to the user in
the instance configuration or the default user. Currently only one
file can be imported at a time.

```
geneos import [TYPE] [FLAGS | NAME [NAME...]] [DEST=]SOURCE [[DEST=]SOURCE...]
```

### Options

```
  -c, --common string   Import into a common directory instead of matching instances.	For example, if TYPE is 'gateway' and NAME is 'shared' then this common directory is 'gateway/gateway_shared'
  -H, --host string     Import only to named host, default is all (default "all")
  -h, --help            help for import
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

