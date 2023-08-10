Renew instance certificates. All matching instances have a new certificate issued using the current signing certificate but the private key file is left unchanged if it exists.

⚠ Warning: While you can renew certificates and keys for `webservers` they will not be used directly as you need to manually import them into the configured truststore/keystore.
