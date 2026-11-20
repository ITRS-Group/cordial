The `init` sub-system is used to initialise your Geneos environment ready to run Geneos instances. To deploy a single instance, for example a Netprobe on a remote endpoint, you should probably use the `geneos deploy` command instead.

On it's own the `init` command will create a new directory structure (or use an existing one if it's considered empty) based on the options you supply. By default it will create a directory named `geneos` in your user's home directory, unless your home directory ends in `geneos` (e.g. `/home/geneos`) in which case it tries that to avoid stuttering in the path. Either specify a directory on the command line or you will be prompted for one. To specify a directory pass it as an argument without a flag. It must be an absolute path.

Bundled component templates, Gateway include files and site configuration files are extracted even when no profile is supplied. No instances are created or releases installed unless a profile or `--restore` is supplied.

It is an error, and the exit status is non-zero, if the directory is not empty and `--force`/`-F` is not given, and nothing is changed. If several profile names are given the directory is only initialised once. Any existing root and signing certificates are always kept and never replaced by `init`, and they are added to the trust chain of the installation.

The installation directory is considered empty if it exists but only contains dot prefixed files and directories - so if you are running the `geneos init` command as a newly created user called `geneos` it should work as expected.

If you want to use an existing directory, including adopting an existing installation, then use the `--force`/`-F` option. The `init` command will create any missing directories and replace bundled component templates, but keep existing instances and site configuration files. See the `init templates` command for how to replace templates separately.

Use `--site`/`-s` to set the initial site name, for example `geneos init --site acme`. Choose a name representing the company, customer, or client using the Geneos deployment, rather than a region or environment; for example, `acme` represents Acme Corp.

Use `--archive`/`-A` to select a release archive or directory on the local machine, or an archive URL. A local path restricts installation to local archives; if it contains no suitable archive, the command does not fall back to downloading. An archive URL is fetched directly.

> [!NOTE]
> Please note the following when choosing and using a site name:
>
> - The site name is currently used for Gateway configuration include directories and filename prefixes, including paths in supplied profiles, but future releases may use it for other features.
> - Spaces become underscores in paths. The resulting name must be a safe single directory path component and cannot conflict with Gateway directories such as `config` or `includes`.
> - If no site name is set, the directory and prefix are `local`.
> - Only one site name is supported per installation.
> - Use the same site name on all Geneos servers for consistent paths.
> - Changing the site name later with `geneos config set` can have unforeseen effects: existing Gateway setups and user profile files are not rewritten. Review them and run `geneos includes install` to populate the new site directory.

## TLS Support

TLS is always enabled on new servers unless the `--insecure` flag is given during initial deployment. By default, new root and signing certificates and private keys are created unless you supply either a signing or a certificate bundle. This typically only makes sense during the initial set-up of Geneos and subsequent deployments should share the trust chain established by the initial deployment.

To deploy on a new server that will share the trust chain with an existing installation, for example a new Netprobe on an endpoint connected to by an existing Gateway, you can use the `--signing-bundle`/`-C`. This bundle should contain the signing certificate and private key used by the existing installation to maintain a consistent trust chain. A new leaf certificate and private key will be generated for the new instance. The root CA in the bundle will automatically be added to the trust chains `ca-bundle.pem` and `ca-bundle.db` files in the Geneos `tls` directory.

The supplied bundle can be either in PEM or PFX/PKCS#12 format.

PEM formatted data containing the required certificates and private key for signing new certificates can be obtained using `geneos tls export` on your existing Geneos server. PEM bundles can be local files, URLs or formatted text prefixed with `pem:`.

PFX/PKCS#12 (file with `.pfx` or `.p12` extensions only) bundles are always protected by a password (which is however considered insecure in the modern world) and this must be supplied with either the `--signing-bundle-password` flag or in an environment variable `ITRS_SIGNING_BUNDLE_PASSWORD`. In both cases the password can be encoded using Coridal expandable format (the output of `geneos aes password`, for example). PFX/PKCS#12 files must be local paths.

## Restoring From Backup

Use `geneos init --restore PATH [DIRECTORY]` (`-R`) to initialise the installation and restore instances and shared files from a `geneos backup` archive. By default, `init` also installs the releases needed by the restored component types. Use `--archive`/`-A` to supply local release archives, or add `--no-install`/`-X` to restore without installing releases.

After archive extraction, missing bundled templates and include files are populated before restored instances are rebuilt. Files from the backup take precedence: existing templates, restored shared configuration files, include files and include links are kept, including with `--force`. Initialization only fills missing files and links; it does not upgrade restored includes to the bundled version.

A backup file will not contain a TLS root or signing certificate and private keys. These must be supplied explicitly using the `--signing-bundle` option (and `--signing-bundle-password` if applicable). Use the `geneos tls export` command to export the necessary certificates and keys when creating a backup.

## Naming Instances

Use `geneos init all --name NAME` (`-n`) to override the default Gateway, licence daemon and Webserver instance names in the `all` profile. This does not rename restored instances. Custom profile entries can use `${option:name}` to consume the option explicitly.

## Adopting An Existing Installation

If you have an existing, now old, Geneos installation that you manage with the command like `gatewayctl`/`netprobectl`/etc. then you can use `geneos` to manage those once you have set the path to the Geneos installation using the `GENEOS_HOME` or `ITRS_HOME` environment variables, or the configuration file.

>[!WARNING]
> `geneos` ignores any changes to the global `.rc` files in your existing installation. You **must** check and adjust individual instance settings to duplicate settings. This can sometimes be very simple, for example if your `netprobectl.rc` files contains a line that sets `JAVA_HOME` then you can set this across all the Netprobes using `geneos set netprobe -e JAVA_HOME=/path/to/java`. More complex changes, such as library paths, will need careful consideration

You can use the environment variable `GENEOS_HOME` pointing to the top-level directory of your installation or set the location in the (user or global) configuration file:

```bash
geneos config set geneos=/path/to/install
```

This is the directory is where the `packages` and `gateway` (etc.) directories live. If you do not have an existing installation that follows this pattern then you can create a fresh layout further below.

Once you have set your directory you check your installation with some basic commands:

```bash
geneos ls     # list instances
geneos ps     # show their running status
geneos show   # show the default configuration values
```

None of these commands should have any side-effects but others will. These may not only start or stop processes but may also convert configuration files to JSON format without prompting. Old `.rc` files are backed-up with a `.rc.orig` extension and can be restored using the `revert` command.
