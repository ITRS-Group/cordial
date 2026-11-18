# `geneos minimal`

A `minimal` Netprobe is one without a Collection agent and associated plugins.

If you will not be using the Collection Agent then this can save about 300MB per download of the release file as well as substantial disk space for the unarchives binaries.

## Usage

```text
geneos minimal
```

### Options

```text
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/geneos/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string           profiles configuration (defaults are ~/.config/geneos/profiles.yaml, /etc/geneos/profiles.yaml)
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
