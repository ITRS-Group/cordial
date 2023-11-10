Move / rename instance(s) `SOURCE` to `DESTINATION`.

If `TYPE` is not given than each component type that has a matching instance `SOURCE` will be moved to `DESTINATION`. If `DESTINATION` is given as an @ followed by a `host` then the instance is moved to the host but the name retained. In all cases, `host` means the name of a pre-defined `geneos host` remote or `@localhost`, not the server hostname.

Additional `KEY=VALUE` parameters can be supplied on the command line to override or supplement instance parameters from the source instance(s). This can be used, for example, to override the listening port of the destination instance. This may not result in expected behaviour if there are multiple instances being moved. Other instance parameters, such as environment variables, should be set on the destination with the `geneos set` command - or removed entirely with the `geneos unset` command.

Any instance using a legacy .rc file is migrated to a newer configuration file format during the move.

The instance is stopped before, and restarted after, the instance is moved.
 
It is an error to try to move an instance to one that already exists with the same name.

If the component support `geneos rebuild` then this is run after the move but before the restart. This allows SANs to be updated as expected.

Moving across hosts is fully supported.
