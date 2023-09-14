Copy instance `SOURCE` to `DESTINATION`. If TYPE is not given than each component type that has a named instance `SOURCE` will be copied to `DESTINATION`. If `DESTINATION` is given as an @ followed by a remote host then the instance is copied to the remote host but the name retained. This can be used, for example, to create a standby Gateway on another host.

Any instance using a legacy .rc file is migrated to a newer configuration file format during the copy.

It is an error to try to copy an instance to one that already exists with the same name on the same host.

The configured port number, if there is one for that TYPE, is updated if the existing one is already in use, otherwise it is left unchanged.

If the component support `rebuild` then this is run after the copy but before the restart. This allows SANs to be updated as expected.
