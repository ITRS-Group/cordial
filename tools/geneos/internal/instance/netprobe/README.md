# `geneos` Netprobes

<https://docs.itrsgroup.com/docs/geneos/current/Netprobe/index.html>

## Configuration

#### `binary`

> Default: `netprobe.linux_64`

The Netprobe program filename. Should not be changed.

#### `home`

> Default: `${GENEOS_HOME}/netprobe/netprobes/NAME`  

This parameter is special in that even though it can be changed it is re-evaluated based on the instance's directory

#### `install`

> Default: `${GENEOS_HOME}/packages/netprobe`
    
The installation directory for Netprobe releases

#### `version`

> Default: `active_prod`

The version of the Netprobe in the the `install` directory above. 

#### `program`

> Default: `${config:install}/${config:version}/${config:binary}`

The full path to the Netprobe executable. The items in the default of the form `${config:NAME}` refer other configuration parameters above.

#### `logdir`

> Default: none

If set, it is used as the directory for the log file below. If not set (the default) then the `home` directory of the instance is used.

#### `logfile`

> Default: `gateway.log`

The file name of the Gateway log file.

#### `port`

> Default: First available from `7036,7100+`

The default port to listen on. The actual default is selected from the first available port in the range defined in `NetprobePortRange` in the program settings.

#### `libpaths`

> Default: `${config:install}/${config:version}/lib64:${config:install}/${config:version}`

This parameter is combined with any `LD_LIBRARY_PATH` environment variable.

#### `autostart`

> Default: `true`

Netprobe instances are set to be started with the default `geneos start` command. Setting `autostart` to false is different to using `geneos disable` to stop an instance from running.

#### `protected`

> Default: `false`


