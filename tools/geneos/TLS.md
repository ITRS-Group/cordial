# `tools/geneos` TLS Certificate Management Changes

This document describes the changes to the way that the `geneos` tools manages TLS certificates for secure communication starting from cordial release v1.26.0.

## Overview

Until the v1.26.0 release, the `geneos` tool did not manage TLS certificates in line with industry best practice. At the time of writing the official Geneos documentation only partly describes the way to create and use certificates, private keys and trusted CA bundles to the various components of Geneos.

The `geneos tls` subsystem commands (and related commands) now create and manage TLS certificates in a more standard and uniform way. This includes:

* **ENABLED BY DEFAULT** - Secure connections between Geneos components using TLS will be enabled by default. As the administrator you can disabled this behaviour using `--insecure` flag on the relevant commands. This only applies to new installations of Geneos. Existing installations that upgrade to v1.26.0 or later will continue work without TLS if it was not already enabled.

* Unless you, as the administrator, provide your own certificates, private keys and trusted CA bundles, the `geneos` tools will create self-signed certificates for you automatically. It will create a root CA certificate, an intermediate signing certificate, instance certificates and private keys for each component that requires one. Connections will be verified using a trusted CA bundle that can include other trusted CA certificates as well as the root CA certificate created by `geneos`. This last feature is important when connecting to external endpoints from Geneos, such as databases, web sites, IAMs for SSO etc.

* All instance certificate files include the leaf certificate as well as the intermediate signing certificate. This is to ensure that clients can validate the full certificate chain. This is commonly referred to with filenames such as `fullchain.pem` in tools like `certbot` and for common web servers like `nginx`.

* A trusted CA bundle (in `${GENEOS_HOME}/tls/ca-bundle.pem`) is used for each instance that requires one. This bundle includes all trusted CA certificates, including the root CA certificate created by `geneos` as well as any other trusted CA certificates you may wish to add.

## Java Keystore and Truststore Support

Geneos components that run under Java in a JVM (i.e. Webserver, standalone Collection Agent and SSO Agent) use Java Keystores and Truststores to manage their TLS certificates. The `geneos` tools can create and manage these keystores and truststores for you automatically. Some Java components, such as Collection Agent plugins, still require PEM formatted certificates and private keys.

When creating or renewing TLS certificates for components that use Java Keystores and Truststores, the `geneos` tools will create the necessary keystore and truststore files, importing the relevant certificates and private keys as needed.

## Commands

### `geneos tls init`

This command initialises the TLS certificate management system for Geneos. It creates a root CA certificate, an intermediate signing certificate, and sets up the necessary directories and files to manage TLS certificates. In previous releases this command would have been used to transition from an insecure to a secure setup. Now, unless you have a specific reason to run it, it is not necessary to run this command as the `geneos` tools will handle TLS certificate management automatically.

### `geneos tls migrate` \[New in v1.26.0\]

This command migrates existing TLS certificates, private keys, and trusted CA bundles from previous versions of Geneos to the new TLS certificate management system. It ensures that existing secure connections continue to work seamlessly after upgrading to v1.26.0 or later.

The steps involved in the migration process include:

* Reordering the contents of the certificate and chain files so that all except root certificates are in a single certificate file.

  Specifically, in earlier releases the `certificate` parameter pointed to a PEM file that contained only the instance leaf certificate and the `chain-file` parameters pointed to a file that contained the intermediate and root CA certificates. These we passed to more geneos components using the `-ssl-certificate` and `-ssl-chain-file` parameters respectively.

  All TLS parameters are now under a `tls` hierarchy, and while `certificate` and `privatekey` have retained their names (but as `tls::certificate` and `tls::privatekey` respectively), the `chain-file` parameter is no longer used and has been replaced by `tls::trusted-roots` instead. The previous `use-chain` parameter is also no longer used and instead a `tls::verify` parameter, with the opposite meaning, has been introduced. 

* Merge any new root CA certificates into the trusted CA bundle.

* Remove previously (incorrectly used) chain files.

* Update instance confirmation parameters to use their new names

### `geneos tls new`

### `geneos tls renew`

The `geneos tls renew` command renews existing TLS certificates that are nearing expiration. It generates new instance certificates signed by the intermediate signing certificate, ensuring that secure connections remain valid. Using this command will also perform the same steps as `geneos tls migrate` for the instances being renewed, if they have not already been migrated.

### `geneos tls inspect` / `geneos tls info` \[New in v1.26.0\]

More detailed information about the TLS certificates whether that are managed by `geneos` or not, can be obtained using this command. The command will support both component/instance matching as well as file paths to certificate files. The level of detail is not as complete as, for example, using `openssl x509 -text -in <certificate-file>`, but it will provide useful information such as subject, issuer, validity period, and SANs.

### `geneos tls list` / `geneos tls ls`

### `geneos tls sync` \[Changed in v1.26.0\]

### `geneos tls export`

The `geneos tls export` command will export combinations of TLS certificates and private keys depending on the flags provided.

### `geneos tls import`

The `geneos tls import` command will import combinations of TLS certificates and private keys depending on the flags provided.

## Examples

## Details
