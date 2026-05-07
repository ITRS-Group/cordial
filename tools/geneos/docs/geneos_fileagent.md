# `geneos fileagent`

A `fileagent` is a File Agent used with Fix Analyser2.

The Fileagent is a small agent used to relay files from one system to a Geneos Netprobe. Normally deployed with Fix Analyser 2, this allows high volume logs to be sent off-host for further processing with minimal impact on the main system which tend to be sensitive to load.

## Usage

```text
geneos fileagent
```

### Options

```text
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
