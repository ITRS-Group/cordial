# `geneos tls` Subsystem Commands

Manage certificates for [Geneos Secure
Communications](https://docs.itrsgroup.com/docs/geneos/current/SSL/ssl_ug.html).

Commands allow for initialisation, create and renewal of certificates as
well as listing details and copying a certificate chain to all other
hosts.

Once initialised then all new instances will also have certificates
created and their configuration set to use secure (encrypted) connections
where possible.

The root and signing certificates are only kept on the local server and
the `sync` command can be used to copy a certificate chain file to remote
servers. Keys, which should be kept secure, are never copied to remote
servers by any commands.

* `geneos tls init`

  Initialised the TLS environment by creating a `tls` directory in
  Geneos and populating it with a new root and intermediate (signing)
  certificate and keys as well as a certificate chain which includes
  both CA certificates. The keys are only readable by the user running
  the command. Also does a `sync` if remotes are configured.

  Any existing instances have certificates created and their
  configurations updated to reference them. This means that any legacy
  `.rc` configurations will be migrated to `.json` files.

* `geneos tls import FILE [FILE...]`

  Import certificates and keys as specified to the `tls` directory as
  root or signing certificates and keys. If both certificate and key are
  in the same file then they are split into a certificate and key and
  the key file is permissioned so that it is only accessible to the user
  running the command.

  Root certificates are identified by the Subject being the same as the
  Issuer, everything else is treated as a signing key. If multiple
  certificates of the same type are imported then only the last one is
  saved. Keys are checked against certificates using the Public Key part
  of both and only complete pairs are saved.

* `geneos tls new [TYPE] [NAME...]`

  Create a new certificate for matching instances, signed using the
  signing certificate and key. This will NOT overwrite an existing
  certificate and will re-use the private key if it exists. The default
  validity period is one year. This cannot currently be changed.

* `geneos tls renew [TYPE] [NAME...]`

  Renew a certificate for matching instances. This will overwrite an
  existing certificate regardless of it's current status of validity
  period. Any existing private key will be re-used. `renew` can be used
  after `import` to create certificates for all instances, but if you
  already have specific instance certificates in place you should use
  `new` above. As for `new` the validity period is a year and cannot be
  changed at this time.

* `geneos tls ls [-a] [-c|-j] [-i] [-l] [TYPE] [NAME...]`

  List instance certificate information. Flags are similar as for the
  main `ls` command but the data shown is specific to certificates.
  Additional flags are:

  * `-a` List all certificates. By default the root and signing
    certificates are not shown
  * `-l` Long list format, which includes the Subject and Signature.
    This signature can be used directly in the Geneos Authentication
    entry for users for non-user authentication using client
    certificates, e.g. Gateway Sharing and Web Server.

* `geneos tls sync`

  Copies certificate chain to all remotes
