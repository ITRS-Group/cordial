Installs the embedded default profile configuration as `profiles.yaml` in the user's Geneos configuration directory (normally `~/.config/geneos` on Linux).

Use `--output` or `-o` to select an existing directory, where the file is named `profiles.yaml`, or to name a `.yaml` or `.yml` file directly. An existing directory takes precedence even if its name ends in a YAML extension.

An identical file is left unchanged. A different file is backed up beside the destination with a timestamp before its extension (for example, `profiles-YYYYMMDDHHMMSS.yaml`). A missing file is created without a backup.

Neither `--profiles` nor a saved `config.profiles` setting changes the destination. `extract` and `update` remain aliases for `install`. See the [profile guide](geneos_profile.md) for profile file locations and usage.