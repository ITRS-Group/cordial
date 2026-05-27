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
