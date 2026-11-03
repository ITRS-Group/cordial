Update the top-level include symlinks to point to files in a versioned include directory on the selected host.

By default, the links are updated to the version embedded in the running `geneos` executable. Use `--version` to select another semantic version already present under `gateway/includes`. If the selected version directory is missing, the command returns an error; the current embedded version is extracted automatically when needed.

Links that already point to the selected version are left unchanged. If a target path already exists with a different link or file, the command returns an error unless `--force` is specified. `--force` removes the existing path before creating the new symlink.

Version values must be valid semantic versions. This prevents a version argument from selecting a path outside the includes directory.