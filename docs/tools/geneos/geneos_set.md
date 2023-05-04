## geneos set

Set instance configuration parameters

### Synopsis


Set configuration item values in global (`geneos set global`), user 
(`geneos set user`), or for a specific instance.

The `geneos set` command allows for the definition of instance properties,
including:
- environment variables (option `-e`)
- for gateways only
  - include files (option `-i`)
- for self-announcing netprobes (san) only
  - gateways (option `-g`)
  - attributes (option `-a`)
  - types (option `-t`)
  - variables (option `-v`)

The `geneos set` command does not rebuild any configuration files 
for instances.  Use `geneos rebuild` for this.

To set "special" items, such as Environment variables or Attributes you should
now use the specific flags and not the old special syntax.

The "set" command does not rebuild any configuration files for instances.
Use "rebuild" to do this.

The properties of a component instance may vary depending on the
component TYPE.  However the following properties are commonly used:
- `binary` - Name of the binary file used to run the instance of the 
  component TYPE.
- `home` - Path to the instance's home directory, from where the instance
  component TYPE is started.
- `install` - Path to the directory where the binaries of the component 
  TYPE are installed.
- `libpaths` - Library path(s) (separated by ":") used by the instance 
  of the component TYPE.
- `logfile` - Name of the log file to be generated for the instance.
- `name` - Name of the instance.
- `port` - Listening port used by the instance.
- `program` - Absolute path to the binary file used to run the instance 
  of the component TYPE. 
- `user` - User owning the instance.
- `version` - Version as either the name of the directory holding the 
  component TYPE's binaries or the name of the symlink pointing to 
that directory.
For more details on instance properties, refer to [Instance Properties](https://github.com/ITRS-Group/cordial/tree/main/tools/geneos#instance-properties).

**Note**: In case for any instance you set a property that is not supported,
that property will be written to the instance's `json` configuration file,
but will not affect the instance.


```
geneos set [flags] [TYPE] [NAME...] [KEY=VALUE...]
```

### Options

```
  -e, --env NAME=VALUE                (all components) Add an environment variable in the format NAME=VALUE
  -i, --include PRIORITY:{URL|PATH}   (gateways) Add an include file in the format PRIORITY:PATH
  -g, --gateway HOSTNAME:PORT         (sans) Add a gateway in the format NAME:PORT
  -a, --attribute NAME=VALUE          (sans) Add an attribute in the format NAME=VALUE
  -t, --type NAME                     (sans) Add a type NAME
  -v, --variable [TYPE:]NAME=VALUE    (sans) Add a variable in the format [TYPE:]NAME=VALUE
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
* [geneos set global](geneos_set_global.md)	 - Set global configuration parameters
* [geneos set host](geneos_set_host.md)	 - Alias for 'host set'
* [geneos set user](geneos_set_user.md)	 - Set user configuration parameters

