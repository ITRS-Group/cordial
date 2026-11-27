Repair broken local include references, and update references to the old `config` directory to site files, in gateway setup files on the selected host. With no names, all gateways are checked. Optional instance names and shell-style glob patterns select gateways just as for `includes list --gateway`; quote patterns to prevent shell expansion. Only matching gateways are changed, but valid include references from all gateways remain available as repair candidates.

Only references with exactly one valid same-basename replacement are changed. HTTP and HTTPS locations, valid files, unmatched names, and ambiguous matches are left unchanged. Candidates are installed shared and versioned includes and valid include references from gateway setups. A top-level active link in `gateway/includes` is preferred over archived version directories. Without an active link, multiple distinct versions of a file make the match ambiguous. Symlinks and paths resolving to the same file count as one candidate.

References to files under `gateway/includes/config` are also changed to the corresponding site file `gateway/<site>/<site>-config.*` (`config.*` gains a `<site>-` prefix; other names and subdirectories are unchanged) but only if that file exists with exactly the same content as the old one. The previous `gateway/<site>/<site>.*` naming can be migrated to `<site>-config.*` on the same condition. Changed or missing site files leave the reference as it is.

Each matching gateway reports its repairs or `no changes made`. Broken references without a unique replacement are reported with the number of matching files and left unchanged; these skips are not command errors.

Command and action auditing follow the standard audit settings. Each setup or saved-configuration write records the Gateway instance, file path, result and a description of each old-to-new include location; setup events also include the backup path where available. Dry runs audit the invocation but do not record file changes.

Replacement locations under the installation's top-level `gateway` directory are relative to the gateway instance directory. Locations outside it are absolute. Matching paths in the instance configuration's `includes` key are updated through the standard instance read/write APIs, including include entries containing pipe-separated labels. Other configuration keys are not inspected or changed. JSON and YAML configurations retain their selected format. Gateway setups are not rebuilt or reloaded.

Use `--dry-run` (`-n`) to list proposed changes without writing files or backups.

```shell
geneos includes fix --dry-run
geneos includes fix
geneos includes fix --dry-run 'prod-*'
geneos includes fix gateway-a gateway-b
```

Existing XML setup files receive history backups. Only the text of the changed include `location` elements is edited in a setup file; comments, whitespace, element order, unknown elements, CDATA sections and the file's encoding and mode are otherwise left exactly as they are. If the file cannot be edited exactly, or does not agree with the parsed setup about the locations to change, the setup is not written and the gateway reports an error. Instance configuration files are replaced atomically without backups, and their formatting may change. Writes to separate files are not transactional; an error on a later write does not roll back an earlier one. The global `--host` (`-H`) option selects the installation.
