Extract the include files embedded in the `geneos` executable to the selected host.

Managed include files are written under `gateway/includes/<version>`, including when the version directory already exists. Existing files that differ are moved to timestamped `.bak` files before replacement. Unchanged files are left in place.

Configuration templates are written under `gateway/includes/config`. Existing configuration templates are never overwritten because they may contain local changes.

Missing top-level include links are created automatically; existing files and links are left alone. Use `--update` to replace existing top-level files and links with links to the extracted version after extraction. If extraction or a filesystem operation fails, the command returns an error; successfully completed earlier operations are not rolled back.
