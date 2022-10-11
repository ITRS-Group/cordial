## geneos aes new

Create a new key file

### Synopsis

Create a new key file. Written to STDOUT by default, but can be
written to a file with the '-k FILE' option.

If the flag '-I' is given then the new key file is imported to the
shared directories of matching components, using '[CRC32].aes' as the
file base name. Currently limited to Gateway and Netprobe types,
including SANs for use by Toolkit 'Secure Environment Variables'.

Additionally, when using the '-I' flag all matching Gateway
instances have the keyfile path added to the configuration and any
existing keyfile path is moved to 'prevkeyfile' to support GA6.x key
file maintenance.

```
geneos aes new [-k FILE] [-I] [TYPE] [NAME...]
```

### Options

```
  -h, --help             help for new
  -H, --host string      Import only to named host, default is all
  -I, --import           Import the keyfile to components and set on matching instances.
  -k, --keyfile string   Optional key file to create, defaults to STDOUT
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos aes](geneos_aes.md)	 - Manage Gateway AES key files

