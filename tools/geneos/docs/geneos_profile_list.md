# `geneos profile list`

List available profiles with their descriptions and the configuration file they came from. The default `column` format prints a readable table. Use `--format` (`-f`) to select `column`, `table`, `csv`, `json`, `pretty`, or `toolkit`. `pretty` outputs indented JSON. If no profile file is selected, `From` is `internal defaults`.
## Usage

```text
geneos profile list
```

### Options

```text
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -f, --format string             Specify the output format: column, csv, json, pretty, table or toolkit (default "column")
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string           profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos profile](geneos_profile.md)	 - Profile
