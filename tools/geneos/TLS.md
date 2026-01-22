# `tools/geneos` TLS Certificate Management Changes

This document describes the changes to the way that the `geneos` tools manages TLS certificates and associated files for secure communication starting from cordial release v1.26.0.

>[!IMPORTANT]
>Please read this document carefully and in full before upgrading to v1.26.0 or later, especially if you are already using TLS certificates for secure communication between Geneos components.
>
>You should ensure you have a backup of your existing Geneos installation before migrating to the new layout. Note that many commands, such as `geneos tls renew` will automatically migrate instances to the new layout as part of their operation.

In most cases it should be enough to simply run:

```bash
geneos backup --shared --tls --aes
geneos tls migrate
geneos restart
```

This will create a `geneos.tar.gz` backup file in the current directory and then migrate your existing TLS certificates to the new layout. Finally it will restart all Geneos components to pick up the changes.

**PLEASE** read the rest of this document at least once before proceeding.

## Overview

Until the v1.26.0 release, the `geneos` tool did not manage TLS certificates in line with industry best practice..

The `geneos tls` subsystem (and other related commands) now create and manage TLS certificates in a more standard and uniform way. The changes includes:

* **ENABLED BY DEFAULT** - Secure connections between Geneos components will be enabled by default. As administrator you can disabled this behaviour using an `--insecure` flag on the relevant commands. This only applies to new installations of Geneos. Existing installations that upgrade to v1.26.0 or later will continue work without TLS if it was not already enabled.

* **COMMAND SYNTAX CHANGES** - Many command have been updated to better support the new TLS layout and options and flags have changed, been renamed or removed completely. Please take time to familiarise yourself with the changes before upgrading to v1.26.0 or later. This applies to not just the `geneos tls` subsystem but also commands like `geneos deploy` and more.

* Unless you, the administrator, provide your own certificates, private keys and trusted CA bundles, `geneos` will create a private PKI for you automatically. It will create a root CA certificate, an intermediate signing certificate, instance certificates and private keys for each component that requires one. Connections will be verified using a trusted CA bundle that can include other trusted CA certificates as well as the root CA certificate created by `geneos`. This last feature is important when connecting to external endpoints from Geneos, such as databases, web endpoints, IAMs for SSO etc.

* All instance certificate files now include the leaf certificate and the intermediate signing certificate. This is to ensure that TLS clients can validate the full trust chain. This is commonly referred to with filenames such as `fullchain.pem` in tools like `certbot` and for common web servers like `nginx`. Note that the root CA certificate is not included in instance certificate files as it is not required for validation by TLS clients as they will only trust root CA certificates that they already have in their trusted CA bundle.

* A single trusted CA bundle (in `${GENEOS_HOME}/tls/ca-bundle.pem`) is used for each instance. This bundle includes all trusted CA certificates, including the root CA certificate created by `geneos` as well as any other trusted CA certificates you may wish to add. This can be overridden on a per-instance basis if required.

* Geneos components acting at clients will verify server certificates by default. This can be disabled on a per-instance basis if required.

* The `geneos tls migrate` command is provided to help you migrate existing TLS certificates, private keys and trusted CA bundles from previous versions of Geneos to the new TLS certificate management system.

## References

The following external website are highly recommended reading for understanding TLS certificate management concepts (but are not specific to Geneos):

* UK National Cyber Security Centre - [Design and build a privately hosted PKI](https://www.ncsc.gov.uk/collection/in-house-public-key-infrastructure)
* Smallstep - [Everything PKI](https://smallstep.com/blog/everything-pki/)

## Java Keystore and Truststore Support

Geneos components that run under Java (`sso-agent` and `webserver` but not the standalone `ca3`) use Java Keystores and Truststores to manage their TLS certificates. The `geneos` tools will create and manage these keystores and truststores for you automatically.

When creating or renewing TLS certificates for components that use Java Keystores and Truststores, the `geneos` tools will create the necessary keystore and truststore files, importing the relevant certificates and private keys as needed.

Existing truststore and keystore files are treated as a source of certificates and private keys during migration. If you have custom certificates or private keys in your existing keystore or truststore files, these will be preserved during migration. Note however that those files will be updated to include the new TLS certificates created by `geneos` as well.

## Commands

The following commands have been changed or added to support the new TLS certificate management system:

### `geneos tls init`

This command initialises the TLS certificate management system for Geneos. It creates a root CA certificate, an intermediate signing certificate, and sets up the necessary directories and files to manage TLS certificates. In previous releases this command would have been used to transition from an insecure to a secure setup. Now, unless you have an existing insecure setup or a specific reason to run it, it is not necessary to run this command as the `geneos` tools will handle TLS certificate management automatically.

If you need to renew the signing certificate use the new option to the `geneos tls renew` command instead.

### `geneos backup`

The `geneos backup` command has been updated to include the TLS CA bundle files in the backup when given both the `--shared` and ``-tls` flags. This ensures that all TLS certificates, private keys, and trusted CA bundles are included in the backup, allowing for a more complete restoration of the Geneos environment if required.

Note that the root CA and signing certificates that are used for leaf certificate creation are not included in the backup as these are considered part of the Geneos installation itself rather than the configuration. To back these up use the `geneos tls export` command but also note that this will not include the root private key for security reasons.

### `geneos tls migrate` - _New in v1.26.0_

This command migrates earlier `geneos` TLS configurations to the new layout and formats. It ensures that existing secure connections continue to work seamlessly after upgrading to v1.26.0 or later.

It is normally enough to simply run `geneos tls migrate` once after upgrading and then restart your instances. You should however ensure you have a backup of your existing TLS files before running the migration, just in case of unexpected behaviour.

The stages in the migration process are:

* `webserver` and `sso-agent` keystores and truststores are imported

  The contents of the files referred to by the component configurations files (`config/security.properties` and `conf/sso-agent.conf`) are imported and converted to PEM format certificate, private key and trusted CA bundle files. This is regardless of whether existing PEM files exist. The `webserver` `config/security.properties` configuration file is updated to separate out the keystore and truststore files and the truststore will refer to the global CA bundle file, but in keystore format.

* Reordering the contents of the certificate and chain files so that all except root certificates are in a single certificate file.

  Specifically, in earlier releases, the `certificate` parameter pointed to a PEM file that contained only the leaf (endpoint) certificate and the `chain-file` parameters pointed to a file that contained the intermediate and root CA certificates. These were passed to Geneos components using the `-ssl-certificate` and `-ssl-certificate-chain` parameters respectively.

* Remove previously (incorrectly used) chain files.

  As the contents of all instance chain files are now included in the instance certificate files, the previous chain files are no longer required and are removed.

* Merge any new root CA certificates into the trusted CA bundle.

  During the above stages, any new root CA certificates that were not already present in the trusted CA bundle are added to it.

* Update instance confirmation parameters to use their new names

  All TLS parameters are now under a `tls` hierarchy, and while `certificate` and `privatekey` have retained their base names (but as `tls::certificate` and `tls::privatekey` respectively), the `chain-file` parameter is no longer used and has been replaced by `tls::ca-bundle` instead. The previous `use-chain` parameter is also no longer used and instead a `tls::verify` parameter, with the opposite meaning, has been introduced. 

### `geneos tls info` / `geneos tls inspect` _New in v1.26.0_

This new command allows you to inspect certificate files and supports PEM, PFX/PKCS#12 and Java keystore formats (including "cacerts" truststores). It displays useful information about the certificates contained within the files, such as subject, issuer, validity period, SANs, fingerprints and more.

### `geneos tls create` _Major changes in v1.26.0_

This command has undergone major changes to provide a more useful features. Not many users knew of this command or used it as the options and output were not very useful.

Changes:

* The output of the command is now a single bundle file in PEM format that contains the private key, the certificate and trust chain _including_ the root CA certificate. This is to ensure that the output can be used directly by TLS clients that may require the full trust chain to validate the certificate.
* The `--days/-D` flag has been replaced by `--expiry/-E` to align with other commands. The value is still specified in days.
* The `--out/-o` flags has been replaced by `--dest/-D` as this should make it clearer that the destination is a directory and not as file.
* The `--bundle/-b` flag has been removed as this is not necessary any more.
* A new `--signer/-S` flag has been added to create a new signer certificate bundle instead if an instance certificate bundle. The signer certificate can then be used on another system to sign instance certificates. The bundle includes the root CA certificate as well as the signer certificate and private key in a format that can be directly ingested by `geneos tls import`.
* The `--san/-s` flag has been removed and replaced by four new flags: `--san-dns/-s`, `--san-ip/-i`, `--san-email/-e` and `--san-uri/-u` to allow more fine-grained control over the Subject Alternative Names (SANs) included in the certificate.

### `geneos tls new`

Changes:

* The `--days/-D` flag has been replaced by `--expiry/-E` to align with other commands. The value is still specified in days.

### `geneos tls renew`

Changes:

* The `--days/-D` flag has been replaced by `--expiry/-E` to align with other commands. The value is still specified in days.

### `geneos tls list` / `geneos tls ls`

The `IsValid` column now shows true only if the certificate can be verified through to a trust root and also that a private key is present and matches the certificate.

### `geneos tls sync`

This command now does something useful, unlike previously.

### `geneos tls export` _Major changes in v1.26.0_

The `geneos tls export` command has been rebuilt from the ground up. Please review the new options and behaviour.

### `geneos tls import` _Major changes in v1.26.0_

The `geneos tls import` command has been rebuilt from the ground up. Please review the new options and behaviour.
