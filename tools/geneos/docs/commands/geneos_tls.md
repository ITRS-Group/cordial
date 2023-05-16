## geneos tls

Manage certificates for secure connections

### Synopsis


Manage certificates for [Geneos Secure Communications](https://docs.itrsgroup.com/docs/geneos/current/SSL/ssl_ug.html).

Sub-commands allow for initialisation, create and renewal of
certificates as well as listing details and copying a certificate
chain to all other hosts.


```
geneos tls [flags]
```

### Options inherited from parent commands

```
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
```

### SEE ALSO

* [geneos](geneos.md)	 - Control your Geneos environment
* [geneos tls import](geneos_tls_import.md)	 - Import root and signing certificates
* [geneos tls init](geneos_tls_init.md)	 - Initialise the TLS environment
* [geneos tls ls](geneos_tls_ls.md)	 - List certificates
* [geneos tls new](geneos_tls_new.md)	 - Create new certificates
* [geneos tls renew](geneos_tls_renew.md)	 - Renew instance certificates
* [geneos tls sync](geneos_tls_sync.md)	 - Sync remote hosts certificate chain files

