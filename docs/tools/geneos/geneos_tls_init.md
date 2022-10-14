## geneos tls init

Initialise the TLS environment

### Synopsis

Initialise the TLS environment by creating a self-signed
root certificate to act as a CA and a signing certificate signed
by the root. Any instances will have certificates created for
them but configurations will not be rebuilt.


```
geneos tls init
```

### Options

```
  -h, --help   help for init
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -q, --quiet           quiet mode
```

### SEE ALSO

* [geneos tls](geneos_tls.md)	 - Manage certificates for secure connections

