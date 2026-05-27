# `geneos tls`

The `tls` sub-system allows you to manage certificates and associated resources for [Geneos Secure Communications](https://docs.itrsgroup.com/docs/geneos/current/SSL/ssl_ug.html).

You can import and manage your own certificates or create your own certificates with your own certificate authority (also known, incorrectly, as "self-signed" certificates).

Commands allow for initialisation, create and renewal of certificates as well as listing details and copying a certificate chain to all other hosts.

Each instance typically uses the following parameters:

* `tls::certificate` - the path to a certificate file in PEM format
* `tls::privatekey` - the path to a private key file for the certificate above
* `tls::ca-bundle` - the path to a file containing one or more PEM formatted certificates that form a trust chain
* `tls::verify` - a boolean parameter that controls the use of the chain file above

Those components which may offer TLS support on listening ports will do so if the `tls::certificate` and `tls::privatekey` parameters are defined. The contents of the files are not validated and are passed to the undelying Geneos binaries as-is. If the files do not exist or are not valid then the component will fail to start.

Those components that act as clients and connect to servers, Geneos or otherwise, will validate the connection based on the `tls::ca-bundle` and `tls::verify` settings. If these are not set or the file does not exist then the connection is still established using TLS but is not verified to be using a trusted certificate.

Please refer to the component documentation (e.g. `geneos help gateway`) for more details on how TLS is used for that component.

Java based components, such as `webserver` and `sso-agent` will also support custom paths to keystore/truststore files, which are also supported by `geneos`. See the documentation for those components for more details.


## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos tls create`](geneos_tls_create.md)	 | Create standalone certificates and keys |
| [`geneos tls export`](geneos_tls_export.md)	 | Export certificate bundle including private key |
| [`geneos tls import`](geneos_tls_import.md)	 | Import certificates (signer if no instances specified) |
| [`geneos tls info`](geneos_tls_info.md)	 | Info about certificates and keys |
| [`geneos tls init`](geneos_tls_init.md)	 | Initialise the TLS environment |
| [`geneos tls list / ls`](geneos_tls_list.md)	 | List certificates |
| [`geneos tls migrate`](geneos_tls_migrate.md)	 | Migrate certificates to the new TLS layout |
| [`geneos tls new`](geneos_tls_new.md)	 | Create instance certificates and keys |
| [`geneos tls renew`](geneos_tls_renew.md)	 | Renew instance certificates |
| [`geneos tls sync`](geneos_tls_sync.md)	 | Sync remote hosts certificate chain files |

### Options

```text
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
