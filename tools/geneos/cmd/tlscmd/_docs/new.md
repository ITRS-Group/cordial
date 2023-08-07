The `tls new` command creates new certificates for matching instances. It overwrites existing certificates.

To create new certificates there must be a valid signing certificate and matching private key. These can be created using the `tls init` command or you can import them using `tls import`.

The `tls new` command differs from `tls create` as the latter creates new certificates in your current directory for later use, while this command creates certificates for matching instances and sets the Common Name based on the component type and name for simple identification.
