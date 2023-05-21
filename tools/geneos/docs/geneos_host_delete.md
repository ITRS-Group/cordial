# `geneos host delete`

Delete a remote host configuration

```text
geneos host delete [flags] NAME...
```

Delete the local configuration referring to a remote host.

### Options

```text
  -F, --force   Delete instances without checking if disabled
  -R, --all     Recursively delete all instances on the host before removing the host config
  -S, --stop    Stop all instances on the host before deleting the local entry
```

## SEE ALSO

* [geneos host](geneos_host.md)	 - Remote Host Operations
