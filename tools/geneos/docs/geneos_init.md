# `geneos init`

The `init` sub-system is used to initialise your Geneos environment ready to run Geneos instances. To deploy a single instance, for example a Netprobe on a remote endpoint, you should probably use the `geneos deploy` command instead.

On it's own the `init` command will create a new directory structure (or use an existing one if it's considered empty) based on the options you supply. By default it will create a directory named `geneos` in your user's home directory, unless your home directory ends in `geneos` (e.g. `/home/geneos`) in which case it tries that to avoid stuttering in the path. Either specify a directory on the command line or you will be prompted for one. To specify a directory pass it as an argument without a flag. It must be an absolute path.

The installation directory is considered empty if it exists but only contains dot prefixed files and directories - so if you are running the `geneos init` command as a newly created user called `geneos` it should work as expected.

If you want to use an existing directory, including adopting an existing installation, then use the `--force`/`-F` option. The `init` command will create any missing directories but will not remove or change existing files. If you adopt an existing installation then see the `init templates` command for how to add default template files to the installation for use by new instances.

## TLS Support

TLS is always enabled on new servers unless the `--insecure` flag is given during initial deployment. By default, new root and signing certificates and private keys are created unless you supply either a signing or a certificate bundle. This typically only makes sense during the initial set-up of Geneos and subsequent deployments should share the trust chain established by the initial deployment.

To deploy on a new server that will share the trust chain with an existing installation, for example a new Netprobe on an endpoint connected to by an existing Gateway, you can use the `--signing-bundle`/`-C`. This bundle should contain the signing certificate and private key used by the existing installation to maintain a consistent trust chain. A new leaf certificate and private key will be generated for the new instance. The root CA in the bundle will automatically be added to the trust chains `ca-bundle.pem` and `ca-bundle.db` files in the Geneos `tls` directory.

The supplied bundle can be either in PEM or PFX/PKCS#12 format.

PEM formatted data containing the required certificates and private key for signing new certificates can be obtained using `geneos tls export` on your existing Geneos server. PEM bundles can be local files, URLs or formatted text prefixed with `pem:`.

PFX/PKCS#12 (file with `.pfx` or `.p12` extensions only) bundles are always protected by a password (which is however considered insecure in the modern world) and this must be supplied with either the `--signing-bundle-password` flag or in an environment variable `ITRS_SIGNING_BUNDLE_PASSWORD`. In both cases the password can be encoded using Coridal expandable format (the output of `geneos aes password`, for example). PFX/PKCS#12 files must be local paths.

## Restoring From Backup

The `--restore/-R` option will restore a backup file created with the `geneos backup` command. This will create the directory structure if it does not exist but will not remove or change any existing files. If this directory is not empty then you can also use the `--force/-F` option to force the use of this directory. Unlike the `geneos restore` command, using `geneos init --restore FILE` restores the complete contents of the backup file while rebuilding the instance configuration files to match the new root directory. Just like the `geneos restore` command it does not alter the component configuration files, only the metadata created and managed by the `geneos` command, so you will need to manually check and edit paths in files like `gateway.setup.xml` etc.

The `--restore/-R` option will also install Geneos software releases used by the instances being restored unless the `--no-install/-X` option is specified. You must either provide download credentials using the `--user/-u` option (or run `geneos login` first) or have the packages already downloaded and then pass the containing directory path using the `--archive/-A` flag.

A backup file will not contain a TLS root or signing certificate and private keys. These must be supplied explicitly using the `--signing-bundle` option (and `--signing-bundle-password` if applicable). Use the `geneos tls export` command to export the necessary certificates and keys when creating a backup.

## Adopting An Existing Installation

If you have an existing, now old, Geneos installation that you manage with the command like `gatewayctl`/`netprobectl`/etc. then you can use `geneos` to manage those once you have set the path to the Geneos installation.

> ❗ WARNING
>
> `geneos` ignores any changes to the global `.rc` files in your existing installation. You **must** check and adjust individual instance settings to duplicate settings. This can sometimes be very simple, for example if your `netprobectl.rc` files contains a line that sets `JAVA_HOME` then you can set this across all the Netprobes using `geneos set netprobe -e JAVA_HOME=/path/to/java`. More complex changes, such as library paths, will need careful consideration

You can use the environment variable `ITRS_HOME` pointing to the top-level directory of your installation or set the location in the (user or global) configuration file:

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

## Usage

```text
geneos init [flags] [DIRECTORY]
```

## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos init all`](geneos_init_all.md)	 | Initialise a more complete Geneos environment |
| [`geneos init demo`](geneos_init_demo.md)	 | Initialise a Geneos Demo environment |
| [`geneos init templates / template`](geneos_init_templates.md)	 | Initialise or overwrite templates |

### Options

```text
  -R, --restore PATH                     Restore from backup file PATH
  -X, --no-install                       Don't install any releases after restore
  -l, --log                              Follow logs after starting instance(s)
  -F, --force                            Ignore existing directories and files and overwrite
  -n, --name string                      Use name for instances and configurations instead of the hostname
  -T, --tls                              Create internal certificates for TLS support
  -C, --signing-bundle string            signing bundle in PEM or PFX/PKCS#12 format.
                                         Use a dash ('-') to be prompted for PEM from console.
                                         PFX/PKCS#12 must be files and are identified by the .pfx or .p12 file extension
      --signing-bundle-password SECRET   Password for PFX/PKCS#12 certificate file.
                                         You will be prompted if required and not supplied as an argument.
      --insecure                         Do not create internal certificates for TLS support
  -A, --archive string                   Directory of releases for installation
  -N, --nexus                            Download from nexus.itrsgroup.com. Requires ITRS internal credentials
  -S, --snapshots                        Download from nexus snapshots. Requires -N
  -V, --version VERSION                  Download matching VERSION, defaults to latest. Doesn't work for EL8 archives. (default "latest")
  -u, --username string                  Username for downloads (password prompted)
  -w, --gateway-template string          A gateway template file
  -e, --env NAME=VALUE                   Environment variable for instance start-up
                                         (Repeat as required)
      --header NAME=VALUE                HTTP header in the format NAME=VALUE
                                         (Repeat as required)
      --allow-root                       allow running as root (not recommended)
  -G, --config string                    config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME                    Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## Examples

```bash
geneos init

```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
