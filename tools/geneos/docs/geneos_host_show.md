# `geneos host show`

Show details of remote host configurations. If no names are supplied then all configured hosts are shown.

The output is always unprocessed, and so any values in `expandable` format are left as-is. This protects, for example, SSH passwords from being accidentally shown in clear text.

```text
geneos host show [flags] [NAME...]
```

## SEE ALSO

* [geneos host](geneos_host.md)	 - Remote Host Operations
