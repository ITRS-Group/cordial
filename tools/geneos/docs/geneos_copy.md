# `geneos copy`

Copy instance(s) `SOURCE` to `DESTINATION`.

If `TYPE` is not given than each component type that has a matching instance `SOURCE` will be copied to `DESTINATION`. If `DESTINATION` is given as an @ followed by a `host` then the instance is copied to the host but the name retained. This can be used, for example, to create a standby Gateway on another host. In all cases, `host` means the name of a pre-defined `geneos host` remote or `@localhost`, not the server hostname.

Additional `KEY=VALUE` parameters can be supplied on the command line to override or supplement instance parameters from the source instance(s). This can be used, for example, to override the listening port of the destination instance. This may not result in expected behaviour if there are multiple instances being copied. Other instance parameters, such as environment variables, should be set on the destination with the `geneos set` command - or removed entirely with the `geneos unset` command.

Any instance using a legacy .rc file is migrated to a newer configuration file format during the copy.

It is an error to try to copy an instance to one that already exists with the same name on the same host.

The configured port number, unless explicitly set using a "port=NUMBER" parameters on the command line and if there is one for that `TYPE`, is updated if the existing one is already in use, otherwise it is left unchanged.

If the component supports `geneos rebuild` then this is run after the copy but before the restart. This allows SANs to be updated as expected.

Copying between hosts is fully supported.

```text
geneos copy [TYPE] SOURCE DESTINATION [KEY=VALUE...]
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
