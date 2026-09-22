The `geneos add` command is used to create a new instance of a component. To deploy a component on a new system, without an existing Geneos installation, use the `geneos deploy` or `geneos init` commands.

The options accepted vary by component `TYPE` and are the resulting parameters are stored in a metadata configuration file in the instance directory. The default configuration file format is in JSON format. There may also be support for YAML configuration files in future releases.
	
The new instance can be started immediately if either the `--start`/`-S` or `--log`/`-l` options are used. The latter will also follow the primary and console log files until interrupted, but the underlying instance will continue running.

## TLS Support

TLS is always enabled on Geneos servers unless the `--insecure` flag was given during initial deployment (by either `geneos init` or `geneos deploy`). The `geneos add` command will create a new instance certificate and private key for the instance unless an existing certificate bundle is supplied via the `--certs-bundle`/`-c` option.

To deploy an instance with an externally issued or existing certificate use the `--certs-bundle`/`-c` option. This bundle should contain a full certificate chain, from a root CA to the leaf certificate, and the private key for the leaf certificate. Any root CAs in the bundle will automatically be added to the trust chains `ca-bundle.pem` and `ca-bundle.db` files in the Geneos `tls` directory.

The supplied bundle can be either in PEM or PFX/PKCS#12 format.

PEM formatted data containing the required certificates and private key can be supplied either as a local file, a URL or formatted text prefixed with a `pem:` prefix. The certificate bundle should also contain any parent certificates required for verification. Root certificates are automatically added to the local trust stores `ca-bundle.pem` and `ca-bundle.db` in the Geneos `tls` directory.

PFX/PKCS#12 (file with `.pfx` or `.p12` extensions only) bundles are always protected by a password (which is however considered insecure in the modern world) and this must be supplied with either the `--certs-password` flag or in an environment variable `ITRS_CERTS_PASSWORD`. In both cases the password can be encoded using Coridal expandable format (the output of `geneos aes password`, for example). PFX/PKCS#12 files must be local paths.

## Instance Configuration Options

See the `geneos set` command documentation (`geneos help set`) for details of the common parameters supported by most components and the documentation for each component type for type-specific options (e.g. `geneos help gateway`).

File can be imported, as with the `geneos import` command, using one or more `--import`/`-I` options. The syntax is the same as for `import` but unlike the command the source can just be a plain file name without a `./` prefix (used in the `geneos import` command to distinguish a file name from an instance name).

The underlying package used by each instance is referenced by a `basename` parameter which defaults to `active_prod`. You can run multiple components of the same type but of different releases. You can do this by configuring additional base-names in advance with `geneos package update` and then by setting the base name with the `--version`/`-V` option.

For a `TYPE` that supports key files a new one is created unless supplied using the `--keyfile` or `--keycrc` options. The `--keyfile` option uses the file given while the `--keycrc` sets the key file path to a key file with the value given (with or with the `.aes` extension). 

Gateways, SANs and Floating probes are given a configuration file based on the templates configured for the different components. The default template can be overridden with the `--template`/`-T` option specifying the source to use. The source can be a local file, a URL or `STDIN`.

Any additional command line arguments are used to set configuration values. Any arguments not in the form `NAME=VALUE` are ignored. Note that `NAME` must be a plain word and must not contain dots (`.`) or double colons (`::`) as these are used as internal delimiters. No component uses hierarchical configuration names except those that can be set by the options above.

You can select the type of SAN or Floating Netprobe using the special syntax for the `NAME` in the form `TYPE:NAME`. The only supported `TYPE`s at the moment, in addition to the default `netprobe`, are `minimal` and `fa2` allowing you to deploy a minimal Netprobe or a Fix Analyser 2 based SAN or Floating Netprobe.
