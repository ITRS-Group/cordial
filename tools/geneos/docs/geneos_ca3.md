# `geneos ca3`

A `ca3` instance is an unmanaged Collection Agent. The instances uses the standard Netprobe installation package and needs Java 17 installed. Releases after 7.1 require Java 21.

A new `ca3` instance is created using local package configuration files, therefore the same package version must be installed locally as on any
remote host.

Component specific parameters:

| parameter         | default                       | description                                           |
| ----------------- | ----------------------------- | ----------------------------------------------------- |
| plugins           | HOME/collection_agent/plugins | Plugin directory, relative to instance home directory |
| health-check-port | 9136                          |                                                       |
| tcp-reporter-port | 7137                          |                                                       |
| minheap           | 512M                          | Java minimum memory                                   |
| maxheap           | 512M                          | Java maximum memory                                   |

## Usage

```text
geneos ca3
```

### Options

```text
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
