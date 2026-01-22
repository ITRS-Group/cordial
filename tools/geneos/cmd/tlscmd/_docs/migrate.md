# `geneos tls migrate`

Migrate existing Geneos TLS certificates and related files to the updated layout and naming conventions.

IMPORTANT: This command modifies existing files and configurations and does not have a "revert" or roll-back option. It is strongly recommended to back up your existing TLS files before running this command. Use the `geneos backup` command to create a backup.

Without a component TYPE or instance NAMEs specified, this command will migrate all Geneos components with TLS configurations.

The following steps are performed during migration:

1. Existing instance certificate and chain files are loaded and merged into a single PEM file. Any trust roots are written to the global `cs-bundle.pem` file. NO VALIDATION of certificates or private keys is performed during this step, unlike other TLS commands.

2. For Java based `sso-adapter` and `web-server` components, Java KeyStores (JKS) and TrustStores are used as the primary source of certificates and private keys. These are then converted to PEM format for consistency with other components. Existing PEM files are ignored for these components.

3. The instance parameters are updates to use the new names and the old parameters are deleted.

The instance parameters are updated as follows:

| Old Parameter         | New Parameter              | Des                                                                     |
| --------------------- | -------------------------- | ----------------------------------------------------------------------- |
| `certificate`         | `tls::certificate`         | Path to the instance certificate file                                   |
| `privatekey`          | `tls::privatekey`          | Path to the instance private key file                                   |
| `certchain`           | `tls::ca-bundle`           | Path to the CA bundle file, defaults to the global `ca-bundle.pem` file |
| `usechain`            | `tls::verify`              | If set to false then no certificate verification is performed           |
| `truststore`          | `tls::truststore`          | Path to the truststore file (for webserver only)                        |
| `truststore-password` | `tls::truststore-password` | Password for the truststore file (for webserver only)                   |
| n/a                   | `tls::minimumversion`      | Sets the minimum TLS version for connections                            |

Each component type will use the above parameters as appropriate. For example, the `truststore` and `truststore-password` parameters are only applicable to the `webserver` component.
