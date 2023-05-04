## geneos add

Add a new instance

### Synopsis


Add a new instance of a component TYPE with the name NAME. The
details will depends on the component TYPE and are saved to a
configuration file in the instance directory. The instance directory
can be found using the `geneos home TYPE NAME` command.

The default configuration file format and extension is `json`. There will
be support for `yaml` in future releases for easier human editing.
	
Gateways and SANs are given a configuration file based on the templates
configured for the different components.


```
geneos add [flags] TYPE NAME
```

### Examples

```

geneos add gateway EXAMPLE1
geneos add san server1 --start -g GW1 -g GW2 -t "Infrastructure Defaults" -t "App1" -a COMPONENT=APP1
geneos add netprobe infraprobe12 --start --log

```

### Options

```
  -T, --template string               template file to use instead of default
  -S, --start                         Start new instance(s) after creation
  -l, --log                           Run 'logs -f' after starting instance. Implies -S to start the instance
  -b, --base string                   select the base version for the instance, default active_prod (default "active_prod")
  -p, --port uint16                   override the default port selection
  -k, --keyfile string                use an external keyfile for AES256 encoding
  -C, --crc string                    use a keyfile (in the component shared directory) with CRC for AES256 encoding
  -e, --env NAME=VALUE                (all components) Add an environment variable in the format NAME=VALUE
  -i, --include PRIORITY:[PATH|URL]   (gateways) Add an include file in the format PRIORITY:[PATH|URL]
  -g, --gateway HOSTNAME:PORT         (sans) Add a gateway in the format NAME:PORT
  -a, --attribute NAME=VALUE          (sans) Add an attribute in the format NAME=VALUE
  -t, --type NAME                     (sans) Add a gateway in the format NAME:PORT
  -v, --variable [TYPE:]NAME=VALUE    (sans) Add a variable in the format [TYPE:]NAME=VALUE
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
* [geneos add host](geneos_add_host.md)	 - Alias for `host add`

