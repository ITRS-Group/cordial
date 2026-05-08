# `geneos package list`

List the packages for the matching TYPE or all component types if no TYPE is given. The `-H` flags restricts the check to a specific remote host.

All timestamps are displayed in UTC to avoid filesystem confusion between local summer/winter times in some locales.

Versions are listed in descending order for each component type, i.e. `latest` is always the first entry for each component.

## Usage

```text
geneos package list [flags] [TYPE]
```

### Options

```text
  -j, --json            Output JSON
  -i, --pretty          Output indented JSON
  -c, --csv             Output CSV
  -t, --toolkit         Output Toolkit formatted CSV
  -I, --instances       Show instances using each version in toolkit mode, using '#' as a separator
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos package](geneos_package.md)	 - Package Operations
