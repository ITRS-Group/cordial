## geneos update

Update the active version of Geneos software

### Synopsis

Update the symlink from the default base name of the package to
the best match for VERSION. The default base directory is 'active_prod'
and is normally linked to the latest version of a component type in the
packages directory. VERSION can either be a semantic version style name or
(the default if not given) 'latest'.

If TYPE is not supplied, all supported component types are updated to VERSION.

Update will stop all matching instances of the each type before
updating the link and starting them up again, but only if the
instance uses the same basename.

The matching of VERSION is based on directory names of the form:

[GA]X.Y.Z

Where X, Y, Z are each ordered in ascending numerical order. If a
directory starts 'GA' it will be selected over a directory with the
same numerical versions. All other directories name formats will
result in unexpected behaviour. If multiple installed versions
match then the lexically latest match will be used. The chosen
match may be much higher than that given on the command line as
only installed packages are used in the search.

If a basename for the synlink does not already exist it will be created,
so it important to check the spelling carefully.


```
geneos update [FLAGS] [TYPE] [VERSION]
```

### Examples

```

geneos update gateway -b active_dev 5.11
geneos update
geneos update netprobe 5.13.2

```

### Options

```
  -b, --base string   Base name for the symlink, defaults to active_prod (default "active_prod")
  -H, --host string   Apply only on remote host. "all" (the default) means all remote hosts and locally (default "all")
  -R, --restart       Restart all instances that may have an update applied
  -h, --help          help for update
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

