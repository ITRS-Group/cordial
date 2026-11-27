# `geneos profile install`

Installs the embedded default profile configuration as `profiles.yaml` in the user's Geneos configuration directory (normally `~/.config/geneos` on Linux).

Use `--output` or `-o` to select an existing directory, where the file is named `profiles.yaml`, or to name a `.yaml` or `.yml` file directly. An existing directory takes precedence even if its name ends in a YAML extension.

An identical file is left unchanged. A different file is backed up beside the destination with a timestamp before its extension (for example, `profiles-YYYYMMDDHHMMSS.yaml`). A missing file is created without a backup.

Neither `--profiles` nor a saved `config.profiles` setting changes the destination. `extract` and `update` remain aliases for `install`. See the [profile guide](geneos_profile.md) for profile file locations and usage.

## Usage

```text
geneos profile install
```

### Options

```text
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/geneos/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
  -o, --output string             Write to an existing directory or a .yaml/.yml file (default: user Geneos configuration directory)
      --profiles string           profiles configuration (defaults are ~/.config/geneos/profiles.yaml, /etc/geneos/profiles.yaml)
```

## SEE ALSO

* [geneos profile](geneos_profile.md)	 - Profile
