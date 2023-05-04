## geneos aes new

Create a new key file

### Synopsis


Create a new key file. Written to STDOUT by default, but can be
written to a file with the `-k FILE` option.

If the flag `-I` is given then the new key file is imported to the
shared directories of matching components, using `CRC32.aes` as the
file base name, where CRC32 is an 8 digit hexadecimal checksum to
help distinguish keyfiles. Currently limited to Gateway and Netprobe
types, including SANs, for use by Toolkit Secure Environment
Variables.

Additionally, when using the `-I` flag any matching Gateway instances
have any existing `keyfile` path setting moved to the `prevkeyfile`
setting to support GA6.x key file rolling.


```
geneos aes new [flags] [TYPE] [NAME...]
```

### Options

```
  -b, --backup string    Backup existing keyfile with extension given (default ".old")
  -D, --default          Save as user default keyfile (will NOT overwrite without -f)
  -H, --host string      Import only to named host, default is all
  -I, --import           Import the keyfile to components and set on matching instances.
  -k, --keyfile string   Optional key file to create, defaults to STDOUT. (Will NOT overwrite without -f)
  -f, --overwrite        Overwrite existing keyfile
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos aes](geneos_aes.md)	 - Manage Geneos compatible key files and encode/decode passwords

