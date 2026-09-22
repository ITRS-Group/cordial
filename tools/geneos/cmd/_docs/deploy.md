Deploy a new instance of component `TYPE`.

The difference between `deploy` and `add` or `init` commands is that `deploy` will check and create the Geneos directory hierarchy as required, then download and/or install packages for the component type and add the instance, optionally starting it.

This allows you to create an instance without having to worry about initialising the set-up and so on. The name if the instance can be given on the command line as `NAME` but defaults to the hostname of the system.

There are many options and which you use depends on any existing Geneos installation, whether you have Internet access and which component you are deploying.

The stages that deploy goes through will help you choose the options you need:

1. For local deployments, if there is no `GENEOS_HOME` (either in the user configuration file or as an environment variable) and a directory is given with `--geneos`/`-D` then a new Geneos installation hierarchy is created and your configuration file is created or updates with the new home directory. If the `--geneos`/`-D` option is given it will override any other setting.

   If the destination for the deployment is a configured remote host then the GENEOS_HONE path configured for that host is always used and the `--geneos`/`-D` option will result in an error if the path is different to the one configured for the remote.

2. If an existing release is installed for the component `TYPE` and a base link (set with `--base`/`-b`, defaulting to `active_prod`) is present then this is used, otherwise `deploy` will install the release selected with the `--version`/`-V` option (default `latest`) either from the official download site or from a local archive. If `--archive`/`-A` is a directory then it is searched for a suitable release archive using the standard naming convention for downloads. If you need to install from a specific file that does not conform to the normal naming conventions then you can override the TYPE and VERSION with the `--override`/`-O` option.

   Please note that if there is already an instance installed but using a separate version then the base link will **NOT** be updated automatically. The release will be downloaded and installed but you will have to also perform a `geneos update` to ensure that other instances are restarted in a controlled way.

3. If the `TYPE` uses templates and the default ones do not exist then they are created.

4. An instance is added with the various options available, just like the `add` command, with the options selected and additional parameters given as `NAME=VALUE` pairs on the command line.

5. If the `--start`/`-S` or `--log`/`-l` options are given then the new instance is started and the logs tail-ed for the latter.

You can select the distribution of SAN or Floating Netprobe using the special syntax for the `NAME` in the form `TYPE:[NAME]`. The `TYPE` can be one of `fa2` or `minimal` to change the package used. `NAME` REMAINS optional, and if not given defaults to the local hostname, but remember to include the colon (`:`) to indicate you are using this syntax.

When an instance is started it has an environment made up of the variables in it's configuration file and some necessary defaults, such as `JAVA_HOME`. Additional variables can be set with the `--env`/`-e` option, which can be repeated as many times as required.

File can be imported, just like the `import` command, using one or more `--import`/`-I` options. The syntax is the same as for `import` but because the import source cannot be confused with the `NAME` of the instance using `deploy` then source can just be a plain file name without the `./` prefix.

The underlying package used by each instance is referenced by a `basename` parameter which defaults to `active_prod`. You can run multiple components of the same type but different releases. You can do this by configuring additional base names in advance with `geneos package update` and then by setting the base name with the `--base`/`-b` option.

Any additional command line arguments are used to set configuration values. Any arguments not in the form `NAME=VALUE` are ignored. Note that `NAME` must be a plain word and must not contain dots (`.`) or double colons (`::`) as these are used as internal delimiters. No component uses hierarchical configuration names except those that can be set by the options above.

## TLS Support

TLS is always enabled on new servers unless the `--insecure` flag is given during initial deployment. By default, new root and signing certificates and private keys are created unless you supply either a signing or a certificate bundle. This typically only makes sense during the initial set-up of Geneos and subsequent deployments should share the trust chain established by the initial deployment.

To deploy a component on a new server that will share the trust chain with an existing installation, for example a new Netprobe on an endpoint connected to by an existing Gateway, you can use the `--signing-bundle`/`-C`. This bundle should contain the signing certificate and private key used by the existing installation to maintain a consistent trust chain. A new leaf certificate and private key will be generated for the new instance. The root CA in the bundle will automatically be added to the trust chains `ca-bundle.pem` and `ca-bundle.db` files in the Geneos `tls` directory.

To deploy an instance with a previously issued certificate use the `--certs-bundle`/`-c` option. This bundle should contain a full certificate chain from a root CA to the leaf certificate that will be used by the instance and the private key for that certificate. The root CA will automatically be added to the trust chains `ca-bundle.pem` and `ca-bundle.db` files in the Geneos `tls` directory.

In both cases the supplied bundle can be either in PEM or PFX/PKCS#12 format.

PEM formatted data containing the required certificates and private key for signing new certificates can be obtained using `geneos tls export` on your existing Geneos server. If you have been give a certificate and key file from a non-Geneos system then you have to make sure they are in PEM format and you can pass them in using the separate flags. The certificate file should also contain any parent certificates required for verification. PEM bundles can be local files, URLs or formatted text prefixed with `pem:`.

PFX/PKCS#12 (file with `.pfx` or `.p12` extensions only) bundles are always protected by a password (which is however considered insecure in the modern world) and this must be supplied with either the `--certs-password` flag or in an environment variable `ITRS_CERTS_PASSWORD`. In both cases the password can be encoded using Coridal expandable format (the output of `geneos aes password`, for example). PFX/PKCS#12 files must be local paths.

## AES Key Files

For a `TYPE` that supports key files have one created unless one is supplied via the `--keyfile` or `--keycrc` options. The `--keyfile` option uses the file given while the `--keycrc` sets the key file path to a key file with the value given (with or with the `.aes` extension).

See the `add` command for more details about other, less used, options.

## Centralised Config Support

To deploy a Gateway instance that supports app keys for authentication you can do something like this:

```bash
$ geneos aes new -S gateway
keyfile 03CA5FA1.aes saved to gateway shared directory on localhost
$ geneos aes encode -A gatewayHub /tmp/app.key
$ geneos deploy gateway central1 -I /tmp/app.key -x "-port 7103" --keyfile ${HOME}/.config/geneos/keyfile.aes gateway-hub=https://hub.example.com:8081 app-key=app.key setup='none'
```
