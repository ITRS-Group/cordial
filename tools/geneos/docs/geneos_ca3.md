# `geneos ca3`

A `ca3` instance is an unmanaged Collection Agent. The instances uses the standard Netprobe installation package and needs Java 17 installed. Releases after 7.1 require Java 21.

A new `ca3` instance is created using the package configuration files from the installed package on the instance's host.

Component specific parameters:

| parameter         | default                       | description                                                                     |
| ----------------- | ----------------------------- | ------------------------------------------------------------------------------- |
| port              | next free from 9137           | The TCP reporter port, unless `tcp-reporter-port` is set                        |
| plugins           | HOME/collection_agent/plugins | Plugin directory, relative to instance home directory                           |
| health-check-port | 9136                          |                                                                                 |
| tcp-reporter-port | port                          | Override the TCP reporter port, by default the instance `port` is used          |
| xms (was minheap) | 512M                          | Java minimum memory                                                             |
| xmx (was maxheap) | 512M                          | Java maximum memory                                                             |

## Usage

```text
geneos ca3
```

### Options

```text
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string           profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
