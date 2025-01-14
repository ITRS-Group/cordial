A `webserver` is an instance of a Web Dashboard Server.

A new `webserver` instance is created using local package configuration files, therefore the same package version must be installed locally as on any remote host.

`geneos` will manage TLS certificates for the webserver instance. When you create a new instance, assuming your Geneos installation has TLS initialised, a new certificate and private key will be created as for other component types but these will also then be added to the Java keystore referenced in the `config/security.properties` file. If you replace or renew the instance certificate or private key you should then use the `geneos rebuild webserver NAME` command to rebuild the keystore file. If you change the `security.properties` file you may need to manually delete the (old) keystore file before running `rebuild`.

If you use `geneos deploy webserver` you can also provide an externally generated certificate bundle without having to initialise TLS for other components.
