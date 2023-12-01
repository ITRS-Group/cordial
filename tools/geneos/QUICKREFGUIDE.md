# `geneos` Quick Reference Guide

| Command | Example | Description |
| ------- | ------- | ----------- |
| `geneos ps` | | Show all running instances |
|    | `geneos ps gateways` | Show only running Gateways |
|    | `geneos ps 'prd*'` | Show only instances that start with 'prd' - note the use of quotes around the wildcarded name, this is to stop the Linux shell from interpreting the '*' as part of a filename |
| `geneos ls` | | Show all configured instances |
| `geneos start` | | Start one or more instances |
| | `geneos start gateway LDN_PRD1 -l` | Start the Gateway 'LDN_PRD1` and watch the log file (CTRL-C to stop watching the log, this will not affect the running instance) |
| `geneos stop` | | Stop instances |
| `geneos restart` | | Restart instances |
| | `geneos restart san -l -F` | Restart all SAN instances, overriding any protection settings and then watching the resulting log file(s) |

--- 






If your Geneos environment is already up-and-running and the `geneos` command installed, then you can run commands like this:

```bash
$ geneos ls gateways
$ geneos ps
$ geneos tls ls
```

The examples above have no side-effects on your Geneos environment and are there to give you information, but it wouldn't be much of an administration tool if it didn't allow you to so more. Here are some example of controlling and interactive with your Geneos environment:

```bash
geneos start netprobe myProbe
geneos restart gateway PROD1
geneos logs -f gateway
```



## Features

The `geneos` program has a wide range of commands. The names of commands have been chosen to be familiar to most administrators, with aliases built-in to make finding the required function easier. Some commands have been grouped into _sub-systems_.

