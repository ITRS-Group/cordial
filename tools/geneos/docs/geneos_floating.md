# `geneos floating`

A Floating Netprobe used the same installation package as the normal Netprobe but uses a confirmation file to identify remote Gateway(s) to connect to. The primary differences between a Floating Netprobe and a SAN (Self-Announcing Netprobe) is that all the configuration for a Floating Netprobe is kept in the Gateway and that a Floating Netprobe has a one-to-one correspondence with it's set-up.
## Usage

```text
geneos floating
```

### Options

```text
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
