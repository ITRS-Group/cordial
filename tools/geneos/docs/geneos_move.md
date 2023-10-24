# `geneos move`

Move (or rename) instance `SOURCE` to `DESTINATION`. If TYPE is not given than each component type that has a named instance `SOURCE` will be moved to `DESTINATION`. If `DESTINATION` is given as an @ followed by a remote host then the instance is moved to the remote host but the name retained.

Any instance using a legacy .rc file is migrated to a newer configuration file format during the move.

The instance is stopped before, and restarted after, the instance is moved.
 
It is an error to try to move an instance to one that already exists with the same name.

If the component support rebuilding a templated configuration then this is run after the move but before the restart. This allows SANs to be updated as expected.

Moving across hosts is fully supported.

```text
geneos move [TYPE] SOURCE DESTINATION [flags]
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
