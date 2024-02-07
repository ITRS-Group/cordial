# `geneos tls`

The `tls` sub-system allows you to manage certificates for [Geneos Secure Communications](https://docs.itrsgroup.com/docs/geneos/current/SSL/ssl_ug.html).

You can import and manage your own certificates or create your own certificates with your own certificate authority (also known, incorrectly, as "self-signed" certificates).

Commands allow for initialisation, create and renewal of certificates as well as listing details and copying a certificate chain to all other hosts.



## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos tls create`](geneos_tls_create.md)	 | Create standalone certificates and keys |
| [`geneos tls import`](geneos_tls_import.md)	 | Import certificates |
| [`geneos tls init`](geneos_tls_init.md)	 | Initialise the TLS environment |
| [`geneos tls list / ls`](geneos_tls_list.md)	 | List certificates |
| [`geneos tls new`](geneos_tls_new.md)	 | Create instance certificates and keys |
| [`geneos tls renew`](geneos_tls_renew.md)	 | Renew instance certificates |
| [`geneos tls sync`](geneos_tls_sync.md)	 | Sync remote hosts certificate chain files |

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
