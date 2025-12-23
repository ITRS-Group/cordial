# `geneos tls`

The `tls` sub-system allows you to manage certificates for [Geneos Secure Communications](https://docs.itrsgroup.com/docs/geneos/current/SSL/ssl_ug.html).

You can import and manage your own certificates or create your own certificates with your own certificate authority (also known, incorrectly, as "self-signed" certificates).

Commands allow for initialisation, create and renewal of certificates as well as listing details and copying a certificate chain to all other hosts.

Each instance may use the following parameters:

* `certificate` - the path to a certificate file in PEM format
* `privatekey` - the path to a private key file for the certificate above
* `certchain` - the path to a file containing one or more PEM formatted certificates that form a trust chains
* `use-chain` - a boolean parameter that controls the use of the chain file above

Those components which may offer TLS protected services on a listening port will do so if the `certificate` and `privatekey` parameters are defined and point to valid files.

Those components that act as clients and connect to servers, Geneos or otherwise, will validate the connection based on the `certchain` and `use-chain` settings. If these are not set or the file does not exist then the connection is still established using TLS but is not verified to be using a trusted certificate.


## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos tls create`](geneos_tls_create.md)	 | Create standalone certificates and keys |
| [`geneos tls export`](geneos_tls_export.md)	 | Export certificates |
| [`geneos tls import`](geneos_tls_import.md)	 | Import certificates |
| [`geneos tls info`](geneos_tls_info.md)	 | Info about certificates and keys |
| [`geneos tls init`](geneos_tls_init.md)	 | Initialise the TLS environment |
| [`geneos tls list / ls`](geneos_tls_list.md)	 | List certificates |
| [`geneos tls migrate`](geneos_tls_migrate.md)	 | Migrate certificates and related files to the updated layout |
| [`geneos tls new`](geneos_tls_new.md)	 | Create instance certificates and keys |
| [`geneos tls renew`](geneos_tls_renew.md)	 | Renew instance certificates |
| [`geneos tls sync`](geneos_tls_sync.md)	 | Sync remote hosts certificate chain files |
| [`geneos tls trust`](geneos_tls_trust.md)	 | Import trusted certificates |

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
