## geneos show

Show runtime, global, user or instance configuration is JSON format

### Synopsis


Show the runtime or instance configuration. The loaded
global or user configurations can be seen through the show global
and show user sub-commands, respectively.

With no arguments show the full runtime configuration that
results from environment variables, loading built-in defaults and the
global and user configurations.

If a component TYPE and/or instance NAME(s) are given then the
configuration for those instances are output as JSON. This is
regardless of the instance using a legacy .rc file or a native JSON
configuration.

Passwords and secrets are redacted in a very simplistic manner simply
to prevent visibility in casual viewing.


```
geneos show [flags] [TYPE] [NAME...]
```

### Options

```
  -r, --raw   Show raw (unexpanded) configuration values
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment

