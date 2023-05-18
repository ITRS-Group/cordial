# geneos copy

Copy instances

```text
geneos copy [TYPE] SOURCE DESTINATION [flags]
```

## Details

Copy instance SOURCE to DESTINATION. If TYPE is not given than each
component type that has a named instance SOURCE will be copied to
DESTINATION. If DESTINATION is given as an @ followed by a remote
host then the instance is copied to the remote host but the name
retained. This can be used, for example, to create a standby Gateway
on another host.

Any instance using a legacy .rc file is migrated to a newer
configuration file format during the copy.

The instance is stopped before and started after the instance is
copied. It is an error to try to copy an instance to one that already
exists with the same name on the same host.

The configured port number, if there is one for that TYPE, is updated
if the existing one is already in use, otherwise it is left
unchanged.

If the component support Rebuild then this is run after the copy but
before the restart. This allows SANs to be updated as expected.

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
