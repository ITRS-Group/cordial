# TLS Certificate Migration Notes

This guide describes the TLS certificate layout introduced in Cordial v1.26.0. It is intended for administrators migrating an existing Geneos installation from an earlier layout. New deployments use TLS by default; review these migration steps before upgrading an installation that already uses TLS.

> [!IMPORTANT]
> Back up your existing Geneos installation before migrating certificates. Some commands, including `geneos tls renew`, can migrate an instance as part of their operation.

For many pre-v1.26.0 installations, the following commands create a backup, migrate TLS files, and restart Geneos components:

```bash
geneos backup --shared --tls --aes all
geneos tls migrate
geneos restart
```

> [!CAUTION]
> `geneos backup` excludes shared directories, AES key files, and instance certificates and private keys by default. The `--shared`, `--aes`, and `--tls` flags are explicit opt-ins because these files may contain confidential configuration, key material, or private keys. Include them only when needed, and protect the resulting archive.

The backup is written to `geneos.tar.gz` in the current directory. Review the sections below to understand how the migration affects your certificate files and trust configuration.

## Overview

Before v1.26.0, the `geneos` command used a different layout for TLS certificates and related files.

The v1.26.0 changes standardised certificate creation and management:

* **TLS by default:** New installations use secure connections unless you pass `--insecure`. Upgrading an existing installation does not enable TLS if it was not already configured.

* **Command changes:** TLS-related options changed in `geneos tls` and other commands, including `geneos deploy`. Check the current command help before using older scripts.

* **Private PKI:** Unless you provide certificates, private keys, and trusted CA bundles, the `geneos` command creates a root CA, an intermediate signing certificate, and instance certificates and keys where required. The trust bundle can include the private root CA and other trusted authorities used by external services.

* All instance certificate files now include the leaf certificate and the intermediate signing certificate. This is to ensure that TLS clients can validate the full trust chain. This is commonly referred to with filenames such as `fullchain.pem` in tools like `certbot` and for common web servers like `nginx`. Note that the root CA certificate is not included in instance certificate files as it is not required for validation by TLS clients as they will only trust root CA certificates that they already have in their trusted CA bundle.

* A single trusted CA bundle (in `${GENEOS_HOME}/tls/ca-bundle.pem`) is used for each instance. This bundle includes all trusted CA certificates, including the root CA certificate created by `geneos` as well as any other trusted CA certificates you may wish to add. This can be overridden on a per-instance basis if required.

* Geneos components acting at clients will verify server certificates by default. This can be disabled on a per-instance basis if required.

* The `geneos tls migrate` command is provided to help you migrate existing TLS certificates, private keys and trusted CA bundles from previous versions of Geneos to the new TLS certificate management system.

## Protect private keys

PEM is an encoding format, not encryption. X.509 certificates and certificate chains contain public information and can be shared with peers that need to verify a connection. A PEM file or bundle may also contain a private key; anyone who obtains that key may be able to impersonate the component while its certificate is trusted. Check bundle contents before exporting or sharing them, restrict private-key access to the Geneos service account, and protect backups that contain keys. If a private key is exposed, replace it and update affected instances and trust configurations.

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

Use `--tls` to include instance certificate and private key files. When backing up component shared directories, use `--shared` as well. These flags are not enabled by default so that backups do not export sensitive files without an explicit decision.

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

  All TLS parameters are now under a `tls` hierarchy, and while `certificate` and `privatekey` have retained their base names (but as `tls.certificate` and `tls.privatekey` respectively), the `chain-file` parameter is no longer used and has been replaced by `tls.ca-bundle` instead. The previous `use-chain` parameter is also no longer used and instead a `tls.verify` parameter, with the opposite meaning, has been introduced. 

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
