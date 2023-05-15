## geneos import

Import files to an instance or a common directory

### Synopsis


Import one or more files to matching instance directories, or with
`--common` flag to a component shared directory. This can be used to
add configuration or license files or scripts for gateways and
netprobes to run. The SOURCE can be a local path, a URL or a `-` for
stdin. PATH is local pathname ending in either a filename or a
directory separator. Is SOURCE is `-` then a destination PATH must be
given. If PATH includes a directory separator then it must be
relative to the instance directory and cannot contain a parent
reference `..`.

Only the base filename of SOURCE is used and if SOURCE contains
parent directories these are stripped and if required should be
provided in PATH.

**Note**: To distinguish a SOURCE from an instance NAME any file in
the current directory (without a `PATH=` prefix) **MUST** be prefixed
with `./`. Any SOURCE that is not a valid instance name is treated as
SOURCE and no immediate error is raised. Directories are created as required.
If run as root, directories and files ownership is set to the user in
the instance configuration or the default user.

Currently only files can be imported and if the SOURCE is a directory
then this is an error.

Like other commands that write to the file system, it can safely be
run as root as the destination directory and file will be changed to
be owned by either the instance or the default user, with the caveat
that any intermediate directories created above the destination
directory (e.g. the first two in `my/long/path`) will be owned by
root.


```
geneos import [flags] [TYPE] [NAME...] [PATH=]SOURCE...
```

### Examples

```

geneos import gateway example1 https://example.com/myfiles/gateway.setup.xml
geneos import licd example2 geneos.lic=license.txt
geneos import netprobe example3 scripts/=myscript.sh
geneos import san localhost ./netprobe.setup.xml
geneos import gateway -c shared common_include.xml

```

### Options

```
  -c, --common string   Import into a common directory instead of matching instances.	For example, if TYPE is 'gateway' and NAME is 'shared' then this common directory is 'gateway/gateway_shared'
  -H, --host string     Import only to named host, default is all (default "all")
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

