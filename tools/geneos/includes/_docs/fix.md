Repair broken local include references in gateway setup files on the selected host. With no names, all gateways are checked. Optional instance names and shell-style glob patterns select gateways just as for `includes list --gateway`; quote patterns to prevent shell expansion. Only matching gateways are changed, but valid include references from all gateways remain available as repair candidates.

Only references with exactly one valid same-basename replacement are changed. HTTP and HTTPS locations, valid files, unmatched names, and ambiguous matches are left unchanged. Candidates are installed shared and versioned includes and valid include references from gateway setups. Symlinks and paths resolving to the same file count as one candidate. Multiple distinct versions of a file make the match ambiguous.

Each matching gateway reports its repairs or `no changes made`. Broken references without a unique replacement are reported with the number of matching files and left unchanged; these skips are not command errors.

Replacement locations under the installation's top-level `gateway` directory are relative to the gateway instance directory. Locations outside it are absolute. Matching paths in the instance configuration's `includes` key are updated through the standard instance read/write APIs, including include entries containing pipe-separated labels. Other configuration keys are not inspected or changed. JSON and YAML configurations retain their selected format. Gateway setups are not rebuilt or reloaded.

Use `--dry-run` (`-n`) to list proposed changes without writing files or backups.

```shell
geneos includes fix --dry-run
geneos includes fix
geneos includes fix --dry-run 'prod-*'
geneos includes fix gateway-a gateway-b
```

Existing XML setup files receive history backups. Instance configuration files are replaced atomically without backups. XML and instance configuration formatting may change. Writes to separate files are not transactional; an error on a later write does not roll back an earlier one. The global `--host` (`-H`) option selects the installation.