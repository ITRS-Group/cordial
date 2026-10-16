# `geneos version`

The `version` command shows program version details.

With the `--debug`/`-d` option it will also output build information which may be useful when reporting issues or to check for suspected problems in build dependencies.

## Usage

```text
geneos version
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
  -t, --toolkit           toolkit formatted CSV output
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
