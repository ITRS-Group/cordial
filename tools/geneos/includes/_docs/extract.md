Extract the include files embedded in the `geneos` executable to the selected host.

Managed include files are written under `gateway/includes/<version>`. Existing files that differ are moved to timestamped `.bak` files before replacement. Unchanged files are left in place. If the version directory already exists, extraction requires `--force`.

Configuration templates are written under `gateway/includes/config`. Existing configuration templates are never overwritten because they may contain local changes.

Use `--update` to update the top-level links to the extracted version after extraction. `--force` also allows those links to be replaced. If extraction or a filesystem operation fails, the command returns an error; successfully completed earlier operations are not rolled back.
