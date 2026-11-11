# `geneos profile extract`

Extracts the embedded default profile configuration to the user's `geneos/profiles.yaml` file. If the file already matches the embedded defaults, it is left unchanged. If it differs, the existing file is backed up as `profiles-YYYYMMDDHHMMSS.yaml` before the embedded defaults are written. A missing file is created without a backup.

This command always targets the standard user configuration path, even if another profile file is selected with `--profiles`. See the [profile guide](geneos_profile.md) for profile file locations and usage.
## Usage

```text
geneos profile extract
```

### Options

```text
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string           profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos profile](geneos_profile.md)	 - Profile
