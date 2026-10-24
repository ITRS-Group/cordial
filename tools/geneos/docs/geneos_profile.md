# `geneos profile`

A `profile` is a pseudo-component representing a Geneos profile instance.

## Usage

```text
geneos profile
```

## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos profile extract / update`](geneos_profile_extract.md)	 | Extract the embedded profile configuration |
| [`geneos profile list / ls`](geneos_profile_list.md)	 | List available profiles |

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
