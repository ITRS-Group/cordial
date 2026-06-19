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
