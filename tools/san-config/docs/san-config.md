# `san-config`

Generate SAN configurations

## Usage

```text
san-config
```

## Commands

| Command | Description |
|-------|-------|
| [`san-config server`](san-config_server.md)	 | Run server for config request |

### Options

```text
      --config string          path to configuration file
  -H, --host HOSTNAME          create configuration for HOSTNAME
  -l, --logfile file           Write logs to file. Use '-' for console or /dev/null for none (default "san-config.log")
  -N, --nowatch[=true|false]   Do not watch configuration file for changes
  -o, --output FILE            output to FILE
  -T, --type string            override hosttype
```

## SEE ALSO

