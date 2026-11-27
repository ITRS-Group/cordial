The `restore` command restores one or more instances and/or shared directories from an archive created by `geneos backup`.

Use the `--list/-l` flag to show the high-level contents from the archive that match other arguments given.

Without either a component type or at least one instance name or pattern the command will do nothing. If given a component type without instance names all matching instances of that component type will be restored. Without a component type but with instances names, all matching instances, regardless of component type, will be restored. To restore all instances use the name `all`. Instance names can use wildcard patterns in shell (or "glob") format. Using the `--shared/-s` flag to restore any shared files and directories only applies to the matching component type if it given.

Those backup archives that contain a `tls/` directory will have the contents restored, which may overwrite existing files. An instance backup created with `--tls` may contain instance certificates and private keys. It deliberately does not contain the user-wide root or signing certificate and private keys required to create or renew certificates. See [restoring with certificates](#restoring-with-certificates) below.

If an instance name is given in the format `DEST=SRC` then instances in the archive called `SRC` will be restored but renamed `DEST`. In this case `SRC` cannot be a wildcard.

Existing instances will not be overwritten, similarly for shared directories. It is an error if an instance named on the command line is not in the archive, but the other instances that are are still restored.

With `--shared`, symlinks in shared directories are restored as links, including dangling links whose targets remain inside the installation. Existing files and links are kept. Link targets that escape the installation and archive entries that write through symlinked directories are rejected. `geneos init --restore` uses the same checks and preserves restored include links when filling missing bundled files.

Gateway site files are restored from the directory matching the currently configured site, or `gateway/local` when no site is configured. Set the same site as the backup before restoring, including when using `geneos init --restore`.

The command accepts a combination of filenames and instance name patterns, with optional renaming, and distinguishes them by validating the arguments. Any arguments that are not valid instance names (or wildcard or rename patterns) are treated as archive files. In case your archive file matches a valid instance name you should either use an absolute path to the file or a `./` prefix.

To read from STDIN use `-` but then you must also specify the compression type used with the `--decompress/-z` flag. Supported values are `gzip`, `bzip2` and `none`.

When restoring an instance any changes to the paths in the instance configuration will be updated to match the destination host's `geneos` root. No component specific files will be changed, including `gateway.setup.xml` and so on.

## Restoring with certificates

When you restore a backup that contains instance certificates, those instances have their original instance certificates restored. However, the backup does **not** contain:

* The user-wide root CA certificate and private key

* The user-wide signing certificate and private key

This is intentional. Root and signing material is managed separately from instance backups, so private signing keys can be controlled centrally rather than duplicated into backup archives. A backup can still contain public trust roots in the installation `tls/` directory and, with `geneos backup --tls`, instance certificates and private keys.

**When restoring to a fresh system without the original signing bundle:**

`geneos init` warns before initialization. If the restore is continued without a signing bundle, it creates a new local root and signing chain. That chain does not match the chain that signed the restored instance certificates. The restored certificates can continue to be used while they remain valid and trusted, but:

* Certificates subsequently renewed with the new local signer will not be trusted by other servers in the Geneos deployment

**To properly restore with certificate trust:**

**Option 1 (recommended): Import the original signing bundle**

1. On the original or central server where the signing keys are held:

   ```bash
   geneos tls export --output signing-bundle.pem
   ```

2. Transfer the exported bundle to the new server

3. When restoring on the new server:

   ```bash
   geneos init --restore backup.tar.gz --signing-bundle <exported-bundle>
   ```

   Use `--signing-bundle -` to paste a PEM bundle on standard input. In an interactive terminal, `geneos` prints the paste prompt and accepts the PEM data through the final newline and Ctrl-D. This can be combined with other interactive prompts because prompts and diagnostics are written to standard error.

   Alternatively, restore after acknowledging the warning, then import the signer:

   ```bash
   geneos init --restore backup.tar.gz
   geneos tls import --file <exported-bundle>
   geneos tls renew
   ```

   `geneos tls renew` runs on the restored server and replaces existing instance certificates using the imported signing key. Select specific component types or instance names when only some certificates need renewing.

**Option 2: Create replacement instance bundles centrally**

If the original signing bundle is unavailable but a trusted central server remains authoritative:

1. On that central/original server, run `geneos tls create` for each restored instance to create an instance certificate bundle.

2. Transfer each bundle to the restored server.

3. Import it for the matching restored instance, for example:

   ```bash
   geneos tls import TYPE NAME --file <instance-bundle>
   ```

In a non-interactive session, the restore continues after logging the warning. If no original signing bundle is supplied or imported, use the central replacement-bundle workflow before relying on renewed certificates to be trusted by the rest of the deployment.
