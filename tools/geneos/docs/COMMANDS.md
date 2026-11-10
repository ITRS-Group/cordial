# Geneos Commands

# `geneos`

The `geneos` program will help you manage your Geneos environment.

The program will help you initialise a new installation, migrate an old `geneos-utils` one, install and update software releases, add and remove instances, control processes and build template based configuration files for SANs and more.

The program works best on Linux but is built for both Windows and MacOS. In the latter two cases it is primarily for the remote management (see below) of Linux instances. There is no support at this time for managing local Windows or MacOS instances of Geneos components.

Most commands work on "instances" of "components". As the names suggest, a "component" is a type of Geneos component such as a Gateway or a Netprobe. An "instance" is a configured instance of a component, a specific Gateway and so on. For a list of components see the "Registered Component Types" below. Each component has it's own help which you can read using `geneos gateway help` etc.

Instances can be created locally or on remote hosts over SSH connections without the need to install `geneos` on the remote Linux server.

Instance names are in the form `[TYPE]:NAME[@HOST]`, where the `[...]` mean that part is optional. The `TYPE` is only used to select the underlying type of Netprobe, e.g. Fix Analyser or plain, for Self-Announcing and Floating Netprobe components during deployment. The `HOST` part is the name of a configured remote host (which may not be the hostname); see the `host` sub-system help with `geneos host help` for more information.

For many commands you can also use wildcards for the `NAME` part. These wildcards are not complex regular expressions but instead follow more common file system patterns. (Note that the exact patterns support are the same as for the Go [`path.Match`](https://pkg.go.dev/path#Match) function.). These wildcards only work on `NAME` and not the `HOST` part and then only for those commands where they make sense, such as `geneos ls`, `geneos start` and so on.

The subsystems below group related functions together and have their own sub-commands, such as `geneos aes password` and `geneos init demo`. Use `geneos SUBSYSTEM help` to see more or, if you are reading this online you should be able to click through for further information.

## Subsystems

| Command / Aliases | Description |
|-------|-------|
| [`geneos adb / dashboard / dashboards`](#geneos-adb)	 | ADB File Operations |
| [`geneos aes`](#geneos-aes)	 | AES256 Key File Operations |
| [`geneos config`](#geneos-config)	 | Configure Command Behaviour |
| [`geneos host`](#geneos-host)	 | Remote Host Operations |
| [`geneos incident`](#geneos-incident)	 | Commands for working with incidents |
| [`geneos includes / include`](#geneos-includes)	 | Manage includes |
| [`geneos init`](#geneos-init)	 | Initialise a Geneos installation |
| [`geneos package`](#geneos-package)	 | Package Operations |
| [`geneos tls`](#geneos-tls)	 | TLS Certificate Operations |

## Control Instances

| Command / Aliases | Description |
|-------|-------|
| [`geneos reload / refresh`](#geneos-reload)	 | Reload Instance Configurations |
| [`geneos restart`](#geneos-restart)	 | Restart Instances |
| [`geneos start`](#geneos-start)	 | Start Instances |
| [`geneos stop`](#geneos-stop)	 | Stop Instances |

## Inspect Instances

| Command / Aliases | Description |
|-------|-------|
| [`geneos command`](#geneos-command)	 | Show Instance Start-up Details |
| [`geneos home`](#geneos-home)	 | Display Instance and Component Home Directories |
| [`geneos list / ls`](#geneos-list)	 | List Instances |
| [`geneos logs / log`](#geneos-logs)	 | View Instance Logs |
| [`geneos ps / status`](#geneos-ps)	 | List Running Instance Details |
| [`geneos show / details`](#geneos-show)	 | Show Instance Configuration |

## Manage Instances

| Command / Aliases | Description |
|-------|-------|
| [`geneos clean`](#geneos-clean)	 | Clean-up Instance Directories |
| [`geneos copy / cp`](#geneos-copy)	 | Copy instances |
| [`geneos disable`](#geneos-disable)	 | Disable instances |
| [`geneos enable`](#geneos-enable)	 | Enable instance |
| [`geneos move / mv / rename`](#geneos-move)	 | Move instances |
| [`geneos protect`](#geneos-protect)	 | Mark instances as protected |
| [`geneos reset`](#geneos-reset)	 | Reset Instance Directories |

## Configure Instances

| Command / Aliases | Description |
|-------|-------|
| [`geneos add`](#geneos-add)	 | Add a new instance |
| [`geneos backup / save`](#geneos-backup)	 | Backup Instances |
| [`geneos delete / rm`](#geneos-delete)	 | Delete Instances |
| [`geneos deploy`](#geneos-deploy)	 | Deploy a new Geneos instance |
| [`geneos import`](#geneos-import)	 | Import Files To Instances Or Components |
| [`geneos migrate`](#geneos-migrate)	 | Migrate Instance Configurations |
| [`geneos rebuild`](#geneos-rebuild)	 | Rebuild Instance Configurations From Templates |
| [`geneos restore / load`](#geneos-restore)	 | Restore instances from archive |
| [`geneos revert`](#geneos-revert)	 | Revert Migrated Instance Configuration |
| [`geneos set`](#geneos-set)	 | Set Instance Parameters |
| [`geneos unset`](#geneos-unset)	 | Unset Instance Parameters |

## Manage Credentials

| Command | Description |
|-------|-------|
| [`geneos login`](#geneos-login)	 | Enter Credentials |
| [`geneos logout`](#geneos-logout)	 | Remove Credentials |

## Miscellaneous

| Command | Description |
|-------|-------|
| [`geneos snapshot`](#geneos-snapshot)	 | Capture a snapshot of each matching dataview |
| [`geneos version`](#geneos-version)	 | Show program version |

## Component Types

| Command / Aliases | Description |
|-------|-------|
| [`geneos ac2`](#geneos-ac2)	 | Active Console 2 |
| [`geneos ca3`](#geneos-ca3)	 | Collection Agent 3 |
| [`geneos fa2`](#geneos-fa2)	 | Fix Analyser 2 |
| [`geneos fileagent`](#geneos-fileagent)	 | File Agent |
| [`geneos floating`](#geneos-floating)	 | Floating Netprobes |
| [`geneos gateway`](#geneos-gateway)	 | Gateways |
| [`geneos licd`](#geneos-licd)	 | Licence Daemon |
| [`geneos md-gateway`](#geneos-md-gateway)	 | MD Gateway |
| [`geneos minimal`](#geneos-minimal)	 | Minimal Netprobes |
| [`geneos netprobe`](#geneos-netprobe)	 | Netprobes |
| [`geneos profile / profiles`](#geneos-profile)	 | Profile |
| [`geneos router`](#geneos-router)	 | Netprobe Router |
| [`geneos san`](#geneos-san)	 | Self-Announcing Netprobes |
| [`geneos sso-agent`](#geneos-sso-agent)	 | SSO Agent |
| [`geneos tr-gateway`](#geneos-tr-gateway)	 | TR Gateway |
| [`geneos webserver`](#geneos-webserver)	 | Web Dashboard Servers |

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos init demo -u email@example.com -l
geneos ps
geneos restart

```

## SEE ALSO

---

# `geneos ac2`

<sub>[Up: geneos](#geneos)</sub>

An `ac2` instance is an Active Console.

## Usage

```text
geneos ac2
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos adb`

<sub>[Up: geneos](#geneos)</sub>

# Active Dashboard Files

Use `geneos adb` to inspect and update exported Active Console dashboard files. These files normally have an `.adb` extension. They are ZIP archives that contain dashboard definitions, resources, and optional startup data.

The commands operate on files. They do not connect to a Gateway or change a running dashboard. Import the result into Active Console to use it.

## Select Input Files

```text
geneos adb info [flags] [PATH...]
geneos adb clean [flags] PATH...
```

`dashboard` and `dashboards` are aliases for `adb`. Both subcommands accept multiple file paths. Quote paths that contain spaces. The shell can expand a file pattern; the commands do not search directories recursively.

```bash
geneos adb info /opt/itrs/dashboards/application.adb
geneos adb info /opt/itrs/dashboards/*.adb
```

No Geneos installation directory is required for these subcommands. `clean` requires at least one input file. With no input, `info` produces no report.

## Inspect a Dashboard

`info` reads the `.Activedashboard` entries inside each archive. It does not change the input file.

| Option | Default | Use |
| --- | --- | --- |
| `--show-xpaths`, `-X` | `false` | Show sorted, distinct data-item XPaths instead of element totals. |

Without `--show-xpaths`, the report lists Gateway, Probe, Managed Entity, Attribute, Sampler, Type, and Dataview names found in supported data-item references. Each heading shows the number of distinct names. Each name has an occurrence count. For Attributes, the report also shows distinct values and their counts. A lack of name for an element is shown as `<no name>`. An empty Type name appears as `<empty>`, while the lack of a Type on a Sampler is not recorded.

1. Get a summary of one dashboard:

   ```bash
   geneos adb info /opt/itrs/dashboards/application.adb
   ```

2. Show its data-item paths:

   ```bash
   geneos adb info --show-xpaths /opt/itrs/dashboards/application.adb
   ```

3. Use the names and paths to select the changes needed before you run `clean`.

Counts and XPath lists accumulate across files in one invocation. For independent reports, run `info` once per file. The report is not a full inventory of all dashboard objects or URL types.

## Choose Output Handling

Choose the output location before you clean a dashboard.

| Option | Default | Use |
| --- | --- | --- |
| `--output DIRECTORY`, `-o DIRECTORY` | No output directory | Write cleaned files to this directory. The command creates the directory if required. Each output uses the input file's base name. |

Without `--output`, `clean` renames each source file to a timestamped backup and writes the cleaned file at the original path. For example, a backup can be named `application-20261102143000.adb`. The timestamp format is `YYYYMMDDhhmmss`.

If the output directory is different from the source directory, the source is not renamed and no source backup is made. If both directories are the same, the command uses the in-place backup behavior.

Use a separate output directory for the first run. Avoid an existing destination with the same name: it can be replaced. Inputs from different directories with the same base name also share one output path. Use separate output directories for those files.

There is no dry-run option. Review the cleaned copy before you replace or import the original.

## Clean a Dashboard

`clean` updates supported `dataitem(...)` references in modifier expressions, severity modifiers, and Geneos URL links. It processes `.Activedashboard` entries in the archive. It copies other archive entries, except `.dbo` startup-data entries, which it removes by default.

With no action options, cleanup has these defaults:

| Element | Default action |
| --- | --- |
| Gateway name | Remove the name predicate. |
| Probe name | Remove the name predicate. |
| Managed Entity name | Remove the name predicate. |
| Managed Entity Attributes | No Attribute action. |
| Sampler name | Keep the name. |
| Sampler Type | Remove explicitly empty Type predicates. Keep non-empty Types and leave absent Types unchanged. See [Type actions](#set-type-actions). |
| Dataview name | Keep the name. |

Removing a name predicate broadens the data-item path. It does not delete the dashboard object or the XPath element. The command also clears the stored URL alias in processed modifier records.

1. Inspect the source with `geneos adb info`.
2. Write a cleaned copy:

   ```bash
   geneos adb clean \
     --output /opt/itrs/dashboards/cleaned \
     /opt/itrs/dashboards/application.adb
   ```

3. Inspect the output:

   ```bash
   geneos adb info --show-xpaths /opt/itrs/dashboards/cleaned/application.adb
   ```

4. Import the cleaned copy into Active Console. Check that its references select the intended data items and that its links and modifiers work.

## Select Processing Options

All the following options are boolean. Without a value, the flag enables the option. Use `--OPTION=false` to disable it explicitly.

| Option | Default | Use |
| --- | --- | --- |
| `--keep-all`, `-k` | `false` | Change the default Gateway, Probe, and Managed Entity name actions to `keep`. Explicit actions for those elements still take precedence. |
| `--skip-modifiers` | `false` | Leave data-item expression and severity modifier records unprocessed. Process links normally. |
| `--skip-links` | `false` | Leave Geneos URL link records unprocessed. Process modifiers normally. |
| `--keep-startup-data` | `false` | Copy `.dbo` entries into the output archive instead of removing them. |

Do not combine `--skip-modifiers` and `--skip-links`; the command rejects that combination.

`--keep-all` does not make a no-change copy. It does not disable Type processing, preserve startup data, or prevent modifier alias cleanup. Use `--keep-startup-data` when startup data must remain. Use `--type keep` to retain explicitly empty Type predicates as well as non-empty Types.

## Set Name Actions

These flags accept an action string. Repeat a flag to apply several actions to that element, in the order given. The first explicit action replaces that flag's built-in default; it is not appended after the default cleanup.

| Option | Element | Default |
| --- | --- | --- |
| `--gateway ACTION` | Gateway name | `remove-all` |
| `--probe ACTION` | Probe name | `remove-all` |
| `--entity ACTION` | Managed Entity name | `remove-all` |
| `--sampler ACTION` | Sampler name | `keep` |
| `--dataview ACTION` | Dataview name | `keep` |

The plural flag names `--gateways`, `--probes`, `--entities`, `--samplers`, and `--dataviews` are aliases for the corresponding singular flags.

| Action | Result |
| --- | --- |
| `keep` | Leave the name at its current value. It does not undo a preceding action. |
| `remove-all` | Remove the name predicate from every selected reference. |
| `remove:PATTERN` | Remove names that match the glob pattern. Leave other names unchanged. |
| `set:NAME` | Set every selected name to `NAME`. |
| `replace:/PATTERN/REPLACEMENT/` | Apply a Go regular expression to the current name and replace every match. |

Glob patterns are not regular expressions: `*` matches any sequence of non-slash characters, `?` matches one, and `[abc]` matches a character from the set. Quote patterns so the shell does not expand them.

Regular-expression replacements use Go syntax. Capture references, such as `${1}`, are supported. Single-quote the action to prevent shell expansion:

```bash
geneos adb clean \
   --output /opt/itrs/dashboards/renamed \
   --gateway 'replace:/^Demo-(.*)$/Production-${1}/' \
   /opt/itrs/dashboards/application.adb
```

For name replacements, the delimiter can be another character, such as `#`. Put it at the start, between the pattern and replacement, and at the end. Escape a delimiter inside either field with a backslash. An invalid replacement expression is logged and leaves that name unchanged.

## Set Type Actions

`--type ACTION` updates the Sampler Type predicate. `--types` is an alias. An explicit empty Type is different from an absent Type predicate.

Repeat the flag to apply actions in order. Each action uses the result of the preceding action.

| Action | Result |
| --- | --- |
| `keep` | Preserve the current Type, including an explicitly empty Type. Leave an absent Type unchanged. |
| `remove-all` | Remove the Type predicate. |
| `remove-empty` | Remove only an explicitly empty Type. Preserve non-empty Types. This is the default. |
| `remove:PATTERN` | Remove Types whose names match a glob pattern. Preserve non-matching Types. |
| `set:NAME` | Set a Type with the given name. |
| `set-empty` | Set an explicitly empty Type predicate. |
| `replace:/PATTERN/REPLACEMENT/` | Replace text in the current Type name with a Go regular expression. Leave an absent Type unchanged. |

For example, `--type set:Demo-System --type 'replace:/^Demo-/Production-/'` produces `Production-System`. After a removal, `keep` does not restore the Type; use `set:NAME` or `set-empty` to create it again. Inspect the output before import.

## Set Attribute Actions

`--attribute ACTION` updates Managed Entity Attribute predicates. `--attributes` is an alias. The default is no Attribute action. Repeat the flag to apply actions in order.

| Action | Result |
| --- | --- |
| `remove-all` | Remove all Managed Entity Attributes in selected references. |
| `remove:NAME` | Remove Attributes whose names match the glob pattern. |
| `remove:NAME=VALUE` | Remove Attributes only when both the name and value match their glob patterns. |
| `add:NAME=VALUE` | Add an Attribute, or replace its value if it already exists. |
| `replace:NAME=VALUE` | Set the value only if this exact Attribute name already exists. |
| `replace:NAME=/PATTERN/REPLACEMENT/` | Apply a Go regular expression to the value of an existing Attribute. |

For Attribute replacements, the regular-expression delimiter must be `/`. Escape a literal `/` inside the pattern or replacement with `\`. Capture references are supported. A replacement value that starts with `/` is treated as a regular-expression specification, not as a literal value.

Attribute actions apply only when the XPath already contains a Managed Entity. They do not create one, so Attribute actions are ignored when the path stops before the Managed Entity. `add:NAME=VALUE` initializes the Attribute map when the existing Managed Entity has no Attributes. To create a Managed Entity explicitly, use `--entity set:NAME` with the Attribute action. The default entity action removes the Managed Entity name predicate; use `--entity keep` when you want to preserve an existing name while changing its Attributes.

For example, preserve an existing Managed Entity name while adding an Attribute:

```bash
geneos adb clean --entity keep --attribute add:Environment=Production input.adb
```

To explicitly add a Managed Entity to a path that does not have one:

```bash
geneos adb clean --entity set:server --attribute add:Environment=Production input.adb
```

## Filter the Changes

Filters select data-item references before actions are applied. They do not select input files or dashboard objects. Without filters, all supported references are eligible.

| Option | Value and use |
| --- | --- |
| `--filter-xpath XPATH` | Parse a simple Geneos XPath as the filter. Glob patterns can be used in its names. This is not an arbitrary XPath query or a glob match against the entire path string. |
| `--filter-gateway PATTERN` | Match Gateway names. |
| `--filter-probe PATTERN` | Match Probe names. |
| `--filter-entity PATTERN` | Match Managed Entity names. |
| `--filter-attribute NAME=VALUE` | Require an Attribute with this exact name and a value that matches the glob pattern. Repeat for several required Attributes. `NAME` alone is treated as `NAME=*`. |
| `--filter-sampler PATTERN` | Match Sampler names. |
| `--filter-type PATTERN` | Match Sampler Type names, subject to the limitations below. |
| `--filter-dataview PATTERN` | Match Dataview names. |

The scalar filters accept one value. Repeating them replaces the earlier value; it does not create an OR list. Repeated Attribute filters with different names require all those Attributes. Repeating the same Attribute name replaces its previous value pattern.

The filter is built in this fixed order, regardless of flag order on the command line:

```text
xpath -> gateway -> probe -> entity -> attributes -> sampler -> type -> dataview
```

Building a component filter can remove lower-level criteria already present in `--filter-xpath`. Existing fields can also retain values from the original XPath rather than taking the separate component flag's value. Prefer either a complete `--filter-xpath` or individual component filters; do not assume that mixing them creates a strict intersection.

For example, this filter uses a complete path to select a Gateway:

```text
--filter-xpath '/geneos/gateway[(@name="Demo Gateway")]'
```

The equivalent simple component filter is `--filter-gateway 'Demo Gateway'`.

Two matching details are important:

- With `--filter-attribute`, also use `--filter-entity '*'` if any Managed Entity name is acceptable. Otherwise, the implicit empty Managed Entity name can prevent named Managed Entities from matching.
- With `--filter-type`, also use `--filter-sampler '*'` if any Sampler name is acceptable. A reference with no Type can still pass a Type filter; inspect the selected paths before setting a Type.

Filters protect the data-item paths of non-matching references. A modifier URL alias is cleared only when at least one matching data-item path in that modifier changes; it is preserved when all paths are filtered out or the selected actions leave them unchanged. Startup data is removed unless `--keep-startup-data` is set.

## Apply Targeted Changes

### Retarget a Gateway

This example replaces a specific Gateway name. It keeps Probe and Managed Entity names, keeps startup data, and explicitly sets the Type to `System` for selected CPU references.

1. Inspect the source with `info --show-xpaths`. Confirm that the CPU references to change should use the `System` Type.
2. Write a separate output copy:

    ```bash
    geneos adb clean \
       --output /opt/itrs/dashboards/retargeted \
       --keep-all \
       --keep-startup-data \
       --filter-gateway 'Demo Gateway' \
       --filter-sampler CPU \
       --gateway 'set:Application Gateway' \
       --type set:System \
       /opt/itrs/dashboards/application.adb
    ```

3. Check the output:

    ```bash
    geneos adb info --show-xpaths /opt/itrs/dashboards/retargeted/application.adb
    ```

4. Import the copy into Active Console and check its data-item references.

### Remove an Environment Attribute from Links

This example selects links with `ENVIRONMENT=Demo`. It removes that Attribute and sets an explicit Type. Modifier records and startup data remain unprocessed.

```bash
geneos adb clean \
   --output /opt/itrs/dashboards/links \
   --keep-all \
   --keep-startup-data \
   --skip-modifiers \
   --filter-entity '*' \
   --filter-attribute ENVIRONMENT=Demo \
   --attribute remove:ENVIRONMENT=Demo \
   --type set:System \
   /opt/itrs/dashboards/application.adb
```

Use this Type only if it is correct for the selected links. Inspect the result as in the previous procedure. To process only modifiers instead, use `--skip-links` rather than `--skip-modifiers`.

### Apply Several Name Actions

This sequence removes a `Demo-` prefix, then changes spaces to underscores in the resulting Gateway name:

```bash
geneos adb clean \
   --output /opt/itrs/dashboards/normalised \
   --gateway 'replace:/^Demo-//' \
   --gateway 'replace:/ /_/' \
   /opt/itrs/dashboards/application.adb
```

Only the Gateway default is replaced here. The other cleanup defaults still apply. Use the processing and action options above to control them.

## Global Options and Help

The following options are inherited from `geneos` and appear in ADB command help:

| Option | Default | Relevance to ADB commands |
| --- | --- | --- |
| `--config FILE`, `-G FILE` | Normal user or system Geneos configuration | Select the main `geneos` configuration file. It is not an ADB input. |
| `--host HOSTNAME`, `-H HOSTNAME` | `all` | Host selection for instance operations. It does not make ADB commands process remote files. |
| `--profiles FILE` | Normal user or system profile configuration | Select profile configuration. ADB commands do not apply profiles. |
| `--allow-root` | `false` | General permission to run as root. Both ADB subcommands already permit root; ordinary file permissions still apply. |

Use a local readable input path and a writable output location. There is no ADB-specific configuration file.

Get the option summary with `--help`. Use the `help` command for the longer embedded command description:

```bash
geneos adb info --help
geneos adb clean --help
geneos help adb clean
```

The generated command references are [adb](#geneos-adb), [info](#geneos-adb-info), and [clean](#geneos-adb-clean). The current implementation is in [cmd/adbcmd]().

## Limits and Fault Isolation

| Symptom or requirement | Check or action |
| --- | --- |
| Input is rejected as a ZIP archive | Check that the file is an exported ADB archive, not extracted dashboard text or an unrelated file renamed to `.adb`. |
| No dashboard details appear | Check that the archive contains an entry with the exact `.Activedashboard` extension. |
| Some paths are unchanged | Check filters and warnings. Only supported simple Geneos paths can be parsed; unsupported paths are skipped. |
| A replacement has no effect | Check delimiters and Go regular-expression syntax. Invalid expressions are logged; unknown actions can be ignored without an error. |
| The output matches more data items than expected | Check removed name, Type, and Attribute predicates. Removing predicates broadens selection. |
| Startup data is missing | Use `--keep-startup-data` if `.dbo` entries must be retained. |
| Modifier aliases change outside the filter | Use `--skip-modifiers` when only links must be processed. Filters do not suppress alias cleanup. |
| Reports for later inputs include earlier names | Run `info` separately for each file. Its counts accumulate within one command invocation. |
| Input or output has the same name as another dashboard | Use separate output directories to prevent base-name collisions. |

Additional limits:

- `info` rejects a `.Activedashboard` entry larger than 10 MiB. Both commands use a line scanner with the normal Go token limit, approximately 64 KiB per line; unusually long records can fail.
- `info` returns on a file-level error. Some record parsing errors are printed or logged while processing continues.
- `clean` logs file-level errors and continues with later inputs. Its command return can still indicate success after individual files fail. Check the logs and confirm that every expected output exists.
- Backup names have one-second timestamp resolution. Repeated in-place runs within one second can collide. Keep a separate backup for important dashboards.
- Name, Type, and Attribute rewriting is not a full validation of Active Console dashboard format or behavior. Always test the output in Active Console before distribution.

## Commands

| Command | Description |
|-------|-------|
| [`geneos adb clean`](#geneos-adb-clean)	 | Clean ADB file(s) |
| [`geneos adb info`](#geneos-adb-info)	 | Info about ADB file(s) |

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos adb clean`

<sub>[Up: geneos adb](#geneos-adb) | [Top: geneos](#geneos)</sub>

Clean and/or update Active Dashboard (ADB) files.

This command is currently applicable only to simple Geneos XPaths used in modifiers and links, which are the most common scenarios. Any Xpath that cannot be parsed successfully will be skipped, and a warning will be logged.

Given one or more Active Dashboard files (normally with an `.adb` extension) this command will apply the specified modifications and create an updated version of each file, by default with the following elements having their name predicates removed: "gateway", "probe" and "entity". With command line flags you can exercise more granular control over which elements to keep, remove, or modify.

The original file is saved by renaming it in the same location as the source file (even when running the command from a different directory) with a timestamp added before the file extension, and the processed file replacing the original. If you specify an output directory using `--output`/`-o`, the processed file will be written there instead, and no backup of the original will be made unless the directory is the same as the one for the source.

If you use the `--keep-all`/`-k` flag, the default action for Gateway, Probe, and Managed Entity names will be `keep`. Type processing still uses `remove-empty` unless overridden. This can be combined with `--filter-*` options to selectively apply the modifications to specific XPaths and modifiers.

To only process modifiers or links, you can use the `--skip-modifiers` and `--skip-links` flags to exclude the respective elements from processing. Note that both flags cannot be used simultaneously, as this would result in no elements being processed.

The `--filter-*` flags allows you to specify either an XPath or specific component matches to selectively apply the cleaning modifications to matching dataitems within the Active Dashboard files. The strings in the filter are applied as "globs", i.e. like shell wildcards. Only the elements matching the filter will be affected by the cleaning options.

For modifier records, the URL alias is cleared only if at least one matching data-item path changes. It is preserved when all paths are filtered out or the selected actions leave them unchanged.

Each filter is applied in the order shown in the command usage, namely: `--filter-xpath`, `--filter-gateway`, `--filter-probe`, `--filter-entity`, `--filter-sampler`, `--filter-dataview`, and `--filter-type`. This order is important because filters are evaluated sequentially, and later filters can further restrict the set of elements matched by earlier filters.

To control which elements have their name predicates removed, use the available command line flags for each element type to specify the elements to keep, remove, or modify. Each option in in the form `--element MODIFIER`, e.g. `--gateway set:GATEWAY_NAME`. Each flag can be repeated as many times as required.

For "gateway", "probe", "entity", "sampler", and "dataview" elements, the following modifications are available:

* `remove-all` - remove the name predicate from the element. This is the default for "gateway", "probe", and "entity" elements.
* `keep` - keep the name predicate for the element. This is the default for "sampler" and "dataview" elements.
* `remove:NAME` - remove the matching name predicate from the element. NAME can be a glob (not regexp) pattern, e.g. "PROD*".
* `set:NAME` - set the name predicates for the element.
* `replace:/pattern/replacement/` - replace the matching pattern, as many times as it appears, with the replacement string. The pattern must be a valid Go regular expression. The replacement can only be a fixed string at this time, and placeholders are not supported. The '/' delimiter can be any character you choose but must be provided at the start and end of the expression as well as a separator between `pattern` and `replacement`. The delimiter can be escaped in either the pattern or the replacement using a backslash.

For "type" elements, the following additional modifications are available:

* `remove-empty` - remove only an explicitly empty Type. Preserve non-empty Types and leave absent Types unchanged. This is the default Type action.
* `set-empty` - set the type to have an empty name. Note that this is different from no type.

Type actions are applied in order to the current result. `keep` preserves both empty and non-empty Types. A non-matching `remove:NAME` preserves the Type. Replacement does not create an absent Type; use `set:NAME` or `set-empty` to create one.

For "attributes" elements, the following modifications are available:

* `remove-all` - remove all attributes from the managed entity element.
* `remove:NAME` - remove specific attributes from the managed entity element.
* `add:NAME=VALUE` - add specific attributes for the managed entity element.
* `replace:NAME=VALUE` - replace the value of the specified attribute with the given value. Unlike `add:NAME=VALUE`, this will only update the value if the attribute already exists.
* `replace:NAME=/PATTERN/REPLACEMENT/` - replace the matching pattern in the attribute value with the replacement string. The pattern must be a valid Go regular expression. The replacement can only be a fixed string at this time, and placeholders are not supported. The delimiter must be a '/' and must appear at the start and end of the expression as well as a separator between `PATTERN` and `REPLACEMENT`. A '/' can be escaped in either the pattern or the replacement using a backslash. Only existing attributes with the specified NAME will be updated.

Attribute actions only apply when the XPath already contains a Managed Entity. They do not create one, so actions are ignored when the path stops before the Managed Entity. Adding an Attribute initializes the map when the Managed Entity exists but has no Attributes. To create a Managed Entity explicitly, use `--entity set:NAME`; use `--entity keep` to preserve an existing Managed Entity name while changing its Attributes.

Modifications are applied in the order given on the command line and there is no priority or precedence among them.

## Usage

```text
geneos adb clean [flags] [PATH...]
```

### Options

```text
  -o, --output directory              Specify the output directory for cleaned ADB files.
                                      If not specified, the cleaned files will replace the
                                      original files in place and a backup of the original
                                      will be created with a timestamp added before the ".adb"
                                      extension.
                                      
      --gateway OPTION                Gateway name updates. Default 'remove-all'
      --probe OPTION                  Probe name updates. Default 'remove-all'
      --entity OPTION                 Entity name updates. Default 'remove-all'
      --sampler OPTION                Sampler name updates. Default 'keep'
      --dataview OPTION               Dataview name updates. Default 'keep'.
                                      
                                      For the above xpath components the valid options, which can be
                                      repeated and are applied in the order given, are:
                                      
                                      'remove-all'          - remove all names
                                      'keep'                - keep all names
                                      'remove:NAME'         - remove names matching NAME. NAME can be
                                                              a glob (not regexp) pattern
                                      'set:ABC'             - set the name to ABC
                                      'replace:/P/R/'       - apply replacement R to names
                                                              matching the regexp pattern R. The
                                                              replacement is applied as many times as
                                                              P matches
                                      
      --type OPTION                   Type updates. Default 'remove-empty'.
                                      
                                      As for xpath components above, but with the addtional options:
                                      
                                      'remove-empty'        - Remove types with an empty name
                                      'set-empty'           - Set types with an empty name. Note that
                                                              this is different from no type.
                                      
      --attribute OPTION              Attribute updates.
                                      
                                      For each attribute, the valid options, which can be repeated and
                                      are applied in the order given, are:
                                      
                                      'remove:NAME[=VALUE]' - remove attribute NAME, optionally only
                                                              those matching VALUE. Both KEY and VALUE
                                                              can be glob (not regexp) patterns
                                      'add:NAME=VALUE'      - add attribute NAME with VALUE
                                      'replace:NAME=VALUE'  - replace attribute NAME with VALUE
                                      'replace:NAME=/P/R/'  - replace attribute NAME with
                                                              the result of applying the regexp
                                                              pattern P to replacement R. The
                                                              replacement is applied as many times as
                                                              P matches
                                      'remove-all'          - remove all attributes
                                      
      --filter-xpath string           Filter for matching XPaths, leaving others unchanged
                                      (applies glob patterns not xpath expressions, e.g. '//managedEntity[(@name="LDN*")]')
      --filter-gateway string         Filter for matching gateways
      --filter-probe string           Filter for matching probes
      --filter-entity string          Filter for matching entities
      --filter-sampler string         Filter for matching samplers
      --filter-type string            Filter for matching types
      --filter-dataview string        Filter for matching dataviews
                                      
                                      All the above filters support glob patterns (not regular expressions).
                                      
      --filter-attribute NAME=VALUE   Filter for matching attributes in the form NAME=VALUE where
                                      VALUE should be '*' if you want to match any value. Can be repeated.
                                      
  -k, --keep-all                      Set default action for all processing actions to 'keep' or do nothing
      --skip-modifiers                Skip processing of modifier elements
      --skip-links                    Skip processing of link elements
      --keep-startup-data             Keep startup data (default is to remove)
      --allow-root                    allow running as root (not recommended)
  -G, --config string                 config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME                 Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string               profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos adb](#geneos-adb)	 - ADB File Operations

---

# `geneos adb info`

<sub>[Up: geneos adb](#geneos-adb) | [Top: geneos](#geneos)</sub>

Show information about the given ADB file(s)

Without options the output lists the number of each element in the ADB file(s). The counts for each value are also shown, and for attributes the counts for each attribute value are displayed.

The `--show-xpaths`/`-X` option displays a list of all XPaths found in the ADB file(s) instead of the counts for each element.
## Usage

```text
geneos adb info [PATH...] [flags]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
  -X, --show-xpaths       Show a list of XPaths rather than totals for element values
```

## SEE ALSO

* [geneos adb](#geneos-adb)	 - ADB File Operations

---

# `geneos add`

<sub>[Up: geneos](#geneos)</sub>

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

## Usage

```text
geneos add [flags] TYPE NAME [KEY=VALUE...]
```

### Options

```text
  -S, --start                         Start new instance after creation
  -l, --log                           Follow the logs after starting the instance.
                                      Implies -S to start the instance
  -p, --port uint16                   Override the default port selection
  -e, --env NAME=VALUE                Environment variable for instance start-up in the format NAME=VALUE
                                      (Repeat as required)
  -V, --version string                Select the version for the instance. Defaults to 'active_prod'
                                      which is the default symlink to the installed release. (default "active_prod")
  -c, --certs-bundle file             Instance certificate bundle file in PEM or PFX/PKCS#12 format.
                                      Use a dash (`-`) to be prompted for data via stdin.
      --certs-password SECRET         Password for PFX/PKCS#12 file decryption.
                                      You will be prompted if not supplied as an argument.
                                      PFX/PKCS#12 files are identified by the .pfx or .p12
                                      file extension and only supported for instance bundles
      --keyfile PATH                  Keyfile PATH
      --keycrc CRC                    CRC of key file in the component's shared "keyfiles" 
                                      directory (extension optional)
  -T, --template PATH|URL|-           Template file to use PATH|URL|-
  -I, --import [DEST=]PATH|URL        import file(s) to instance. DEST defaults to the base
                                      name of the import source or if given it must be
                                      relative to and below the instance directory
                                      (Repeat as required)
  -i, --include PRIORITY:[PATH|URL]   An include file in the format PRIORITY:[PATH|URL]
                                      (Repeat as required, gateway only)
  -g, --gateway HOSTNAME:PORT         A gateway connection in the format HOSTNAME:PORT
                                      (Repeat as required, san and floating only)
  -a, --attribute NAME=VALUE          Attribute in the format NAME=VALUE
                                      (Repeat as required, san only)
  -t, --type NAME                     A type NAME
                                      (Repeat as required, san only)
  -v, --variable [TYPE:]NAME=VALUE    A variable in the format [TYPE:]NAME=VALUE
                                      (Repeat as required, san only)
      --insecure                      Do not create certificates for TLS support.
                                      Ignored if --certs-bundle is given.
      --allow-root                    allow running as root (not recommended)
  -G, --config string                 config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME                 Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string               profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos add gateway EXAMPLE1
geneos add san server1 --start -g GW1 -g GW2 -t "Infrastructure Defaults" -t "App1" -a COMPONENT=APP1
geneos add netprobe infraprobe12 --start --log

```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos aes`

<sub>[Up: geneos](#geneos)</sub>

The `aes` sub-system allows you to manage AES256 keyfiles and perform encryption and decryption.

The `aes` commands provide tools to manage Geneos AES256 key files as documented in the [Secure Passwords](https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm) section of the manuals.

In addition to the functionality built-in to Geneos as described in the Gateway documentation these encoded password can also be included in configuration files so that plain text passwords and other credentials are not visible to users.


## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos aes decode`](#geneos-aes-decode)	 | Decode a Geneos AES256 format password using a key file |
| [`geneos aes encode`](#geneos-aes-encode)	 | Encode password to a Geneos AES256 secret using a key file |
| [`geneos aes list / ls`](#geneos-aes-list)	 | List key files |
| [`geneos aes new`](#geneos-aes-new)	 | Create a new key file |
| [`geneos aes password / passwd`](#geneos-aes-password)	 | Encode a password with an AES256 key file |
| [`geneos aes set`](#geneos-aes-set)	 | Set keyfile for instances |

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos aes decode`

<sub>[Up: geneos aes](#geneos-aes) | [Top: geneos](#geneos)</sub>

The `aes decode` command decodes a Geneos AES256 format password using the key file(s) given.

If an "expandable" string is given with the `--expandable`/`-e` option it must be of the form `${enc:...}` (be careful to single-quote this string when using a shell) and is then decoded using the key file(s) listed and the ciphertext in the value. All other flags and arguments are ignored.

The format of `expandable` strings is documented here:

<https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#readme-expandable-formats>

A specific key file can be given using the `--keyfile`/`-k` flag and an alternative ("previous") key file with the `--previous`/`-v` flag. If either of these key files are supplied then the command tries to decode the given ciphertext and a value may be returned. An error is returned if all attempts fail.

When using the `--expandable`/`-e` or `--keyfile`/`-k` flags you can also use `--raw`/`-r` to output the decoded value without any prefix or newline (unless they are in the secret) for easier use in scripting, e.g. `$(geneos aes decode -k keyfile -p ciphertext)` in bash. This flag is ignored when using keyfiles from instance matches.

If no key files are provided then matching instances are checked for configured key files and each one tried or the default key file paths are tried. An error is only returned if all attempts to decode fail. The ciphertext may contain the optional prefix `+encs+`. If both `-p` and `-s` options are given then the argument to the `--password`/`-p` flag is used. To read a ciphertext from STDIN use `--source -`/`-s -`.

## Usage

```text
geneos aes decode [flags] [TYPE] [NAME...]
```

### Options

```text
  -k, --keyfile KEYFILE     Path to keyfile (default /home/peter/.config/docs/keyfile.aes)
  -v, --previous KEYFILE    Path to previous keyfile (default /home/peter/.config/docs/prevkeyfile.aes)
  -e, --expandable string   The keyfile and ciphertext in expandable format (including '${...}')
  -p, --password string     Geneos formatted AES256 password
  -s, --source string       Alternative source for password
  -r, --raw                 Output raw decoded value for --expandable/-e and --keyfile/-k decoding only (no prefix and no newline if not part of the secret, for scripting)
      --allow-root          allow running as root (not recommended)
  -G, --config string       config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME       Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string     profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
# don't forget to use single quotes to escape the ${...} from shell
# interpolation
geneos aes decode -e '${enc:~/.config/geneos/keyfile.aes:hexencodedciphertext}'

# decode from the environment variable "MY_ENCODED_PASSWORD"
geneos aes decode -e '${enc:~/.config/geneos/keyfile.aes:env:MY_ENCODED_PASSWORD}'

# try to decode using AES key file configured for all instances
geneos aes decode -p +encs+hexencodedciphertext

# try to decode using the AES key file associated with the 'Demo Gateway' instance
geneos aes decode gateway 'Demo Gateway' -p +encs+hexencodedciphertext

```

## SEE ALSO

* [geneos aes](#geneos-aes)	 - AES256 Key File Operations

---

# `geneos aes encode`

<sub>[Up: geneos aes](#geneos-aes) | [Top: geneos](#geneos)</sub>

The `aes encode` command either encodes plaintext to a Geneos AES256 format password using a key file, or creates a Gateway "app key" file.

A key file should be provided using the `-keyfile`/`-k` option for a file path, the `--crc`/`-c` option for the CRC of a shared key file, or otherwise all matching instances that have a configured key file are used to produce an encrypted password.

Without matching `TYPE` or `NAME` the encode command with not update all instances. To force this, use `all` as an explicit wildcard.

## Encoding Passwords

For encoding passwords, the password can be provided in one three ways:

1. The default is to prompt for the plaintext and again to verify they match.
2. Alternatively the password can be provided directly on the command line using the `-p plaintext` flag or,
3. From an external source using the `-s PATH` or `-s URL` option where the contents of the file at PATH ir URL is read and used. If `-s -` is used then the plaintext is read from `STDIN`.

It is important to note that no whitespace is trimmed from the plaintext. This can have unexpected results if you do something like this:

```bash
$ echo "test" | geneos aes encode -s -
```

Rather than:

```bash
$ echo -n "test" | geneos aes encode -s -
```

## App Keys

To create an app key file suitable for connecting to an SSO Agent, Gateway Hub or Obcerv from a Gateway for Centralised Configuration basic authentication support use the `--app-key`/`-A` flag. The value passed with the flag must be a valid provider, which is one of: "`ssoAgent`", "`gatewayHub`" or "`obcerv`". These values are case-sensitive.

The client ID and client secret can either be passed on the command line using the `--client-id`/`-C` and `--client-secret`/`-S` flags respectively, or you will be prompted to enter one or both using a non-echoing password-like dialogue.

The app key file contents are written to STDOUT unless you supply an `-app-key-file`/`-a` filename. This should be a file name and not a file path, and will be used to write an app key file in each matching instance home directory. If you supply a file path then the results are undetermined.

The contents of the app key output of saved file should be identical to that of the `-store-app-key` Gateway command line option but using an external key file (only).

## Usage

```text
geneos aes encode [flags] [TYPE] [NAME...]
```

### Options

```text
  -e, --expandable             Output in 'expandable' format
  -k, --keyfile KEYFILE        Path to keyfile
  -c, --crc string             CRC of existing component shared keyfile to use (extension optional)
  -p, --password SECRET        Password
  -s, --source string          Alternative source for plaintext password
  -o, --once                   Only prompt for password once, do not verify. Normally use '-s -' for stdin
  -A, --app-key PROVIDER       SSO PROVIDER, one of ssoAgent, obcerv, gatewayHub
  -C, --client-id SECRET       Client ID for --app-key, prompted if not set
  -S, --client-secret SECRET   Client Secret for --app-key, prompted if not set
  -a, --app-key-file string    App-key filename, if saving per instance, otherwise defaults to STDOUT
      --allow-root             allow running as root (not recommended)
  -G, --config string          config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME          Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string        profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash

```

## SEE ALSO

* [geneos aes](#geneos-aes)	 - AES256 Key File Operations

---

# `geneos aes list`

<sub>[Up: geneos aes](#geneos-aes) | [Top: geneos](#geneos)</sub>

The `aes list` command lists details of the key files referenced by matching instances.

If given the `--shared`/`-S` flag then the key files in the shared component directory are listed. This can be filtered by host with the `--host`/`-H` and/or by component `TYPE`.

The default output is human-readable table format. You can select CSV or JSON formats using the appropriate flags.

## Usage

```text
geneos aes list [flags] [TYPE] [NAME...]
```

### Options

```text
  -S, --shared            List shared key files
  -j, --json              Output JSON
  -i, --pretty            Output indented JSON
  -c, --csv               Output CSV
  -t, --toolkit           Output Toolkit formatted CSV
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos aes list gateway
geneos aes ls -S gateway -H localhost -c

```

## SEE ALSO

* [geneos aes](#geneos-aes)	 - AES256 Key File Operations

---

# `geneos aes new`

<sub>[Up: geneos aes](#geneos-aes) | [Top: geneos](#geneos)</sub>

The `aes new` command creates a new key file. By default, the contents are written to STDOUT.

To write to a specific file use the `--keyfile`/`-k` option. To write to your user's default key file location use the `--user`/`-u` flag. If both these options are used then the `--user`/`-u` flag takes precedence.

If the `--shared`/`-S` flag is set then the new key file is written to the shared `keyfiles` directory of component `TYPE`, using the base-name of its 8-hexadecimal digit checksum to distinguish it from other key files. In all examples the CRC is shown as `DEADBEEF` in honour of many generations of previous UNIX documentation. There is a very small chance of a checksum clash. If TYPE is not given then all components that support key files are used. When saving key files to shared component directories the contents of the key file are not written to STDOUT, but if combined with `--keyfile/-k` or `--user/-U` then the same key file is written to both places.

To update instances to use the new shared key file, use the `--update`/`-U` option.

An existing non-shared key file with the same name will be backed-up using the suffix given with the `--backup`/`-b` option which defaults to `-prev`. This is only likely to apply to key files being saved to explicit paths with the `--keyfile` or `--user` options. Shared key files use the CRC of the contents as the base name and so are not backed-up as they are very unlikely to clash name-wise.

## Usage

```text
geneos aes new [flags] [TYPE] [NAME...]
```

### Options

```text
  -k, --keyfile KEYFILE   Path to key file, defaults to STDOUT
      --user              Write to user key file (typically "${HOME}/.config/geneos/keyfile.aes")
  -b, --backup string     Backup existing keyfile with extension given (default "-prev")
  -F, --force             Force overwriting an existing key file
  -S, --shared            Write the keyfile to matching component shared directories
  -U, --update            Update keyfile settings on matching instances
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos aes new
geneos aes new -F ~/keyfile.aes
geneos aes new -S gateway

```

## SEE ALSO

* [geneos aes](#geneos-aes)	 - AES256 Key File Operations

---

# `geneos aes password`

<sub>[Up: geneos aes](#geneos-aes) | [Top: geneos](#geneos)</sub>

The `aes password` command encodes a password using the key file in the user's config directory, which by default is `${HOME}/.config/geneos`. If no key file exists it is created. Output is in "expandable" format.

You will be prompted to enter the password (twice, for validation) unless one of the flags is set to select an alternative source for the plaintext.

To encode a plaintext password using a specific key file please use the `geneos aes encode` command

## Usage

```text
geneos aes password [flags]
```

### Options

```text
      --allow-root          allow running as root (not recommended)
  -G, --config string       config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME       Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
  -p, --password SECRET     Password
      --profiles string     profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
  -s, --source PATH|URL|-   External source for password PATH|URL|-
```

## SEE ALSO

* [geneos aes](#geneos-aes)	 - AES256 Key File Operations

---

# `geneos aes set`

<sub>[Up: geneos aes](#geneos-aes) | [Top: geneos](#geneos)</sub>

The `aes set` command imports a key file and sets parameters on matching instances.

To create a key file with new contents use the `aes new` command.

If the `--shared`/`-s` flag is set then the provided key file is imported to matching component shared key file directories and matching instances have their key file parameters set to point to the new location.

If `--crc CRC`/`-c CRC` is given then the 8 hex-digit CRC is used to look up an existing key file in the component's shared key file directory and if found then the matching instances are updated to use this. In this case no key files are changed.

Depending on the `--no-roll`/`-N` flag, any matching instances may have their `prevkeyfile` parameter updated to reference any original key file and, if a new key file is written in a non-shared location the previous file is also renamed using the suffix in the `--backup`/`-b` flag.

Key files are only set on components that support them.

## Usage

```text
geneos aes set [flags] [TYPE] [NAME...]
```

### Options

```text
  -k, --keyfile PATH|URL|-   Key file to use. PATH|URL|-
                             Path to a local file, a URL or a dash for STDIN. (default "-")
  -c, --crc CRC              CRC of an existing shared keyfile to use
  -b, --backup string        Backup any existing keyfile with extension given (default "-prev")
  -N, --no-roll              Do not roll any existing keyfile to previous keyfile setting
  -s, --shared               Set as a shared keyfile, using the CRC as the file name prefix
      --allow-root           allow running as root (not recommended)
  -G, --config string        config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME        Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string      profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos aes](#geneos-aes)	 - AES256 Key File Operations

---

# `geneos backup`

<sub>[Up: geneos](#geneos)</sub>

Backup instance data in an archive file, either for a safety backup or to be restored on another system using the `geneos restore` command.

Each matching instance has its primary configuration files included in the archive. By default, no AES files, instance certificates, or instance private keys are included in the archive (taken from the instance configuration and not by filename pattern). If saving any of these files is enabled using the flags below, then only files in the instance directory are included; files outside the instance directory are never saved. Files that match the clean or purge lists for that type are ignored unless the `--all/-a` flag is given, in which case those files are not skipped (but sensitive files like AES key files and private keys are still only included using the specific options below).

The global `tls/` directory is always included in the backup. This is so that the trusted roots that may be used by instances are included. The directory should never contain anything other than certificates, which are public information.

The default output format is a `tar.gz` file (gzipped tar). You can also select bzip2 or none using the `--compress/-c` option. Unless you set your own archive file name using the `--output/-o` flag, the file name will be automatically generated based on how many unique component types and instance names there are. The `--output/-o` flags will also accept `-` to mean output to stdout.

If a single instance is selected than the default destination archive is made up of `geneos`, the component type and the instance name, in the format `geneos-TYPE-NAME.tar.gz` in the current directory. Use the `--output`/`-o` option to override, including a `-` to indicate STDOUT for piping to another command. In this case any other messages are written to STDERR. If written to a file. any existing file with the same name is overwritten. If the program is invoked using another name then that is used instead of `geneos`. All spaces in instance names are replaced with underscores. So, a Gateway with the name "Demo Gateway" would be saved as `geneos-gateway-Demo_Gateway.tar.gz`.

You can add a time and date to the file name using the `--datetime/-D` flag, and this is in the fixed numeric format like `YYYYMMDDhhmmss`.

If there are no instances selected then the archive is named either simple `geneos.tar.gz` if multiple components match or `geneos-TYPEs.tar.gz` (note the plural of `TYPEs`, e.g. `gateways`) if the instances are all of the same type.

Instances matching across multiple hosts are not supported, and this returns an error. If the instance (or all matching instances) are on the same remote host then these are all saved and the default filename will include the host label (the `geneos host` name).

To include the component shared directory/directories, use the `--shared` option.

To include AES files, use the `--aes` option. Without this option the files that are skipped are those referenced by the instance `keyfile` and `prevkeyfile` parameters. When using `--shared` files with an `.aes` extension or any `keyfile` directory and it's contents are skipped.

To include certificate and private key files, use the `--tls` option. Without this option the files skipped are those referenced by the instance parameters `certificate`, `privatekey` and `certchain`. When using `--shared` all files with the extensions `.pem`, `.key` and `.crt` are skipped.

File over 2MiB are not included in the archive unless the `--all/-a` option is used, but this limit can be controlled using the `--size/-s` option which accepts all the common units, e.g. `50KB` or `10MiB`. Using a size of `0` (zero) removes the limit.

The contents of the archive are relative to the root of the Geneos installation, and the `geneos load` command will refactor any changes to paths in the instance configuration JSON files.

Note that instances are not stopped or restarted as part of the backup process. If files are locked or changed during the backup then the contents may be inconsistent. If you want to stop the instance before backup and restart it afterwards then you can use the `geneos stop` and `geneos start` commands in a script.

## Usage

```text
geneos backup [flags] [all] | [TYPE] [NAME...]
```

### Options

```text
  -o, --output string     Write to "DEST". Without a destination filename the command creates
                          a file in the current directory with a name based on the contents of the
                          archive. If DEST is a directory or has a "/" suffix then the file is
                          written to that directory using the same naming format as if no file was
                          given. Directories are created as required.
  -D, --datetime          Add a datetime string (YYYYMMDDhhmmss) the auto-generated file names
  -z, --compress type     Compression type. One of `gzip`, `bzip2` or `none`. (default "gzip")
  -s, --size string       Skip files larger than this size unless --all is used. Accepts suffixes
                          with common scale units such as K, M, G with both B and iB units,
                          e.g. "2MiB". "0" (zero) means no limit to file sizes. (default "2MiB")
  -A, --all               Include all files except AES key files, certificates and private keys.
      --shared            Include per-component shared directories from outside instance directories.
                          
      --aes               Include AES key files.
      --tls               Include certificates, private keys and certificate chains.
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos backup gateway gw1
geneos backup all

```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos ca3`

<sub>[Up: geneos](#geneos)</sub>

A `ca3` instance is an unmanaged Collection Agent. The instances uses the standard Netprobe installation package and needs Java 17 installed. Releases after 7.1 require Java 21.

A new `ca3` instance is created using local package configuration files, therefore the same package version must be installed locally as on any remote host.

Component specific parameters:

| parameter         | default                       | description                                           |
| ----------------- | ----------------------------- | ----------------------------------------------------- |
| plugins           | HOME/collection_agent/plugins | Plugin directory, relative to instance home directory |
| health-check-port | 9136                          |                                                       |
| tcp-reporter-port | 7137                          |                                                       |
| xms (was minheap) | 512M                          | Java minimum memory                                   |
| xmx (was maxheap) | 512M                          | Java maximum memory                                   |

## Usage

```text
geneos ca3
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos clean`

<sub>[Up: geneos](#geneos)</sub>

Clean the working directory for matching instances.

The default behaviour is to leave the instance running and only remove files known to be old or inactive.

The `--full`/`-F` option has been deprecated, use the `reset` command instead. It will remove more files than the basic clean and restart the instance.

**Note**: Files removed by `geneos clean` are defined in the geneos main configuration file `geneos.json` as `[TYPE].clean`. Files removed by `geneos clean -F` or `geneos clean --full` are defined in the geneos main configuration file `geneos.json` as `[TYPE].purge`. Both these lists are formatted as a PathListSeparator (typically a colon) separated list of file pattern globs (not regular expressions).

## Usage

```text
geneos clean [flags] [TYPE] [NAME...]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
# Delete old logs and config file backups without affecting the running
# instance
geneos clean gateway Gateway1

```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos command`

<sub>[Up: geneos](#geneos)</sub>

Show the command line and environment variables for matching instances.

If given the `--json`/`-j` flag then the output is an array of objects with the details of each instance and the command environment used.

The options `--extras`/`-x` and `--env`/`-e` can be used to add one-off extra command line parameters and environment variables to the command constructors. This is primarily to match the `start` and `restart` commands and for diagnostics and debugging

## Usage

```text
geneos command [TYPE] [NAME...]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -e, --env NAME=VALUE    Extra environment variable (Repeat as required)
  -x, --extras string     Extra args passed to process, split on spaces and quoting ignored
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
  -j, --json              JSON formatted output
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos config`

<sub>[Up: geneos](#geneos)</sub>

# `geneos config`

The config sub-system allow you to control the environment of the `geneos` program itself.

## General Configuration

* `/etc/geneos/geneos.json` - Global options
* `${HOME}/.config/geneos/geneos.json` - User options
* Environment variables ITRS_`option` - where `.` is replaced by `_`, e.g. `ITRS_DOWNLOAD_USERNAME`

General options are loaded from the global file first, then the user file and any environment variables override both files. The currently supported options are:

* `geneos`

  The home directory for all other commands.

  If set, the environment variable `GENEOS_HOME` overrides any settings in save configuration file. This is to maintain backward compatibility with older tools. This value is can also be set with the environment variables `ITRS_HOME` for legacy support.

* `download.url`

  The base URL for downloads for automating installations. Not yet used. If files are locally downloaded then this can either be a `file://` style URL or a directory path.

* `download.username` `download.password`

  These specify the username and password to use when downloading packages. They can also be set as the environment variables, but the environment variables are not subject to expansion and so cannot contain Geneos encoded passwords (see below):

  * `ITRS_DOWNLOAD_USERNAME`
  * `ITRS_DOWNLOAD_PASSWORD`

* `snapshot.username` `snapshot.password`

  Similarly to the above, these specify the username and password to use when taking dataview snapshots. They can also be set as the environment variables, with the same restrictions as above:

  * `ITRS_SNAPSHOT_USERNAME`
  * `ITRS_SNAPSHOT_PASSWORD`

* `GatewayPortRange` & `NetprobePortRange` & `LicdPortRange`


## Commands

| Command | Description |
|-------|-------|
| [`geneos config import`](#geneos-config-import)	 | import Instances |
| [`geneos config set`](#geneos-config-set)	 | Set program configuration |
| [`geneos config show`](#geneos-config-show)	 | Show program configuration |
| [`geneos config unset`](#geneos-config-unset)	 | Unset a global program parameter |

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos config import`

<sub>[Up: geneos config](#geneos-config) | [Top: geneos](#geneos)</sub>

Import one or more instances from a `tar.gz` archive created by `geneos config export`


## Usage

```text
geneos config import [flags] [TYPE] [NAME...]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash

```

## SEE ALSO

* [geneos config](#geneos-config)	 - Configure Command Behaviour

---

# `geneos config set`

<sub>[Up: geneos config](#geneos-config) | [Top: geneos](#geneos)</sub>

Set configuration parameters for the `geneos` program.

Each value is in the form of `KEY=VALUE` where key is the configuration item and value an arbitrary string value. Where a `KEY` is in a hierarchy use a dot (`.`) as the delimiter.

While you can set arbitrary keys only some have any meaning. The most important one is `geneos`, the path to the root directory of the Geneos installation managed by the program. If you change or remove this value you may break the functionality of the program, so please be careful.

For an explanation of the various configuration parameters see the main documentation.

## Usage

```text
geneos config set [KEY=VALUE...] [flags]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos config set geneos="/opt/geneos"

```

## SEE ALSO

* [geneos config](#geneos-config)	 - Configure Command Behaviour

---

# `geneos config show`

<sub>[Up: geneos config](#geneos-config) | [Top: geneos](#geneos)</sub>

The show command outputs the current configuration for the `geneos` program in JSON format. By default it shows the processed values from the on-disk copy of your program configuration and not the final configuration that the running program uses, which includes many built-in defaults.

Using the `--all`/`-a` options then the output is the full running configuration from loading and merging all configuration files that apply, including internal and external defaults.

If any arguments are given then they are treated as a list of keys to limit the output to just those keys that match and have a non-nil value.

No values that are encrypted are shown decrypted.

## Usage

```text
geneos config show [KEY...] [flags]
```

### Options

```text
  -a, --all               Show all the parameters including all defaults
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos config](#geneos-config)	 - Configure Command Behaviour

---

# `geneos config unset`

<sub>[Up: geneos config](#geneos-config) | [Top: geneos](#geneos)</sub>

Unset removes the program configuration value for any arguments given on the command line.

No validation is done and there if you mistype a key name it is still considered valid to remove an non-existing key.

## Usage

```text
geneos config unset [KEY...] [flags]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos config](#geneos-config)	 - Configure Command Behaviour

---

# `geneos copy`

<sub>[Up: geneos](#geneos)</sub>

Copy instance(s) `SOURCE` to `DESTINATION`.

If `TYPE` is not given than each component type that has a matching instance `SOURCE` will be copied to `DESTINATION`. If `DESTINATION` is given as an @ followed by a `host` then the instance is copied to the host but the name retained. This can be used, for example, to create a standby Gateway on another host. In all cases, `host` means the name of a pre-defined `geneos host` remote or `@localhost`, not the server hostname.

Additional `KEY=VALUE` parameters can be supplied on the command line to override or supplement instance parameters from the source instance(s). This can be used, for example, to override the listening port of the destination instance. This may not result in expected behaviour if there are multiple instances being copied. Other instance parameters, such as environment variables, should be set on the destination with the `geneos set` command - or removed entirely with the `geneos unset` command.

Any instance using a legacy .rc file is migrated to a newer configuration file format during the copy.

It is an error to try to copy an instance to one that already exists with the same name on the same host.

The configured port number, unless explicitly set using a "port=NUMBER" parameters on the command line and if there is one for that `TYPE`, is updated if the existing one is already in use, otherwise it is left unchanged.

If the component supports `geneos rebuild` then this is run after the copy but before the restart. This allows SANs to be updated as expected.

Copying between hosts is fully supported.

## Usage

```text
geneos copy [TYPE] SOURCE DESTINATION [KEY=VALUE...]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos delete`

<sub>[Up: geneos](#geneos)</sub>

Delete matching instances.

Instances that are marked `protected` are not deleted without the `--force`/`-F` option, or they can be unprotected using `geneos protect -U` first.

Instances that are running are not removed unless the `--stop`/`-S` option is given.

The instance directory is removed without being backed-up. The user running the command must have the appropriate permissions and a partial deletion cannot be protected against.

If called with no arguments `delete` will take no action. If you do want to match all instances then you must use the explicit instance wildcard `all`.

## Usage

```text
geneos delete [flags] [TYPE] [NAME...]
```

### Options

```text
  -S, --stop              Stop instances first
  -F, --force             Force delete of protected instances
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos deploy`

<sub>[Up: geneos](#geneos)</sub>

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

## Usage

```text
geneos deploy [flags] TYPE [NAME] [KEY=VALUE...]
```

### Options

```text
  -D, --geneos GENEOS_HOME            Installation directory. Prompted if not given and not found
                                      in existing user configuration or environment ${GENEOS_HOME}
  -S, --start                         Start new instance after creation
  -l, --log                           Start created instance and follow logs.
                                      (Implies --start to start the instance)
  -x, --extras string                 Extra args passed to initial start, split on spaces and quoting ignored
                                      Use this option for bootstrapping instances, such as with Centralised Config
  -p, --port port                     Override the default port selection
  -n, --nosave                        Do not save a local copy of any downloads
  -C, --signing-bundle string         signing certificate bundle file, in PEM or PFX/PKCS#12 format.
                                      Use a dash ('-') to be prompted for PEM from console.
                                      PFX/PKCS#12 must be files and are identified by the .pfx or .p12 file extension
  -c, --certs-bundle file             Instance certificate bundle file in PEM or PFX/PKCS#12 format.
                                      Use a dash ('-') to be prompted for PEM from console.
                                      PFX/PKCS#12 must be files and are identified by the .pfx or .p12 file extension
      --certs-password SECRET         Password for PFX/PKCS#12 signing or certificate bundle file.
                                      You will be prompted if required and have not supplied one as an argument.
                                      The password can be in cordial expandable format.
      --insecure                      Do not initialise TLS subsystem.
                                      Ignored if --certs-bundle is given.
      --keyfile PATH                  Keyfile PATH to use. Default is to create one
                                      for TYPEs that support them
      --keycrc CRC                    CRC of key file in the component's shared "keyfiles" 
                                      directory to use (extension optional)
  -u, --username string               Username for downloads
                                      Credentials used if not given.
  -P, --password SECRET               Password for downloads
                                      Prompted if required and not given
  -b, --base string                   Select the base version name for the instance.
                                      Defaults to 'active_prod' which is the default
                                      symlink to the installed release. (default "active_prod")
  -V, --version VERSION               Use this VERSION of package
                                      Doesn't work for EL8/9/10 archives. (default "latest")
  -L, --local                         Install from local archives only
  -A, --archive string                URL or file path to release archive
                                      or a directory to search for local release archives.
                                      Local paths imply --local
  -O, --override [TYPE:]VERSION       Override the [TYPE:]VERSION for archive
                                      files with non-standard names
      --nexus                         Download from nexus.itrsgroup.com
                                      Requires ITRS internal credentials
      --snapshots                     Download from nexus snapshots
                                      Implies --nexus
      --template PATH|URL|-           Template file to use (if supported for TYPE). PATH|URL|-
  -I, --import [DEST=]PATH|URL        import file(s) to instance. DEST defaults to the base
                                      name of the import source or if given it must be
                                      relative to and below the instance directory
                                      (Repeat as required)
  -e, --env NAME=VALUE                Environment variable for instance start-up in the format NAME=VALUE
                                      (Repeat as required)
  -i, --include PRIORITY:[PATH|URL]   An include file in the format PRIORITY:[PATH|URL]
                                      (Repeat as required, gateway only)
  -g, --gateway HOSTNAME:PORT         A gateway connection in the format HOSTNAME:PORT
                                      (Repeat as required, san and floating only)
  -a, --attribute NAME=VALUE          Attribute in the format NAME=VALUE
                                      (Repeat as required, san only)
  -t, --type NAME                     A type NAME
                                      (Repeat as required, san only)
  -v, --variable [TYPE:]NAME=VALUE    A variable in the format [TYPE:]NAME=VALUE
                                      (Repeat as required, san only)
      --header NAME=VALUE             HTTP header in the format NAME=VALUE
                                      (Repeat as required)
      --allow-root                    allow running as root (not recommended)
  -G, --config string                 config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME                 Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string               profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash

```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos disable`

<sub>[Up: geneos](#geneos)</sub>

Mark any matching instances as disabled. The instances are also stopped.

If called with no arguments `disable` will take no action. If you do want to match all instances then you must use the explicit instance wildcard `all`.

## Usage

```text
geneos disable [TYPE] [NAME...] [flags]
```

### Options

```text
  -S, --stop              Stop instances
  -F, --force             Force disable instances
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos enable`

<sub>[Up: geneos](#geneos)</sub>

Enable matching instances and, if the `--start`/`-S` options is set then start the instance. Only those instances that were disabled are started when the `--start`/`-S` flag is used.

If called with no arguments `delete` will take no action. If you do want to match all instances then you must use the explicit instance wildcard `all`.

## Usage

```text
geneos enable [flags] [TYPE] [NAME...]
```

### Options

```text
  -S, --start             Start enabled instances
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos fa2`

<sub>[Up: geneos](#geneos)</sub>

An `fa2` is a Fix-Analyser2 Netprobe
## Usage

```text
geneos fa2
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos fileagent`

<sub>[Up: geneos](#geneos)</sub>

A `fileagent` is a File Agent used with Fix Analyser2.

The Fileagent is a small agent used to relay files from one system to a Geneos Netprobe. Normally deployed with Fix Analyser 2, this allows high volume logs to be sent off-host for further processing with minimal impact on the main system which tend to be sensitive to load.

## Usage

```text
geneos fileagent
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos floating`

<sub>[Up: geneos](#geneos)</sub>

A Floating Netprobe used the same installation package as the normal Netprobe but uses a confirmation file to identify remote Gateway(s) to connect to. The primary differences between a Floating Netprobe and a SAN (Self-Announcing Netprobe) is that all the configuration for a Floating Netprobe is kept in the Gateway and that a Floating Netprobe has a one-to-one correspondence with it's set-up.
## Usage

```text
geneos floating
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos gateway`

<sub>[Up: geneos](#geneos)</sub>

A `gateway` component is an instance of a [Geneos Gateway](https://docs.itrsgroup.com/docs/geneos/current/processing/gateway_introduction/index.html).

## Configuration

The Gateway instance configuration, as opposed to the runtime configuration of the Gateway itself, is stored in the instance configuration file `gateway.json`. This file is created with the instance and updated when the `geneos set` and `geneos unset` commands are used to change parameters. The configuration file is stored in the instance working directory, which is shown when running the `geneos list` command. This file should not be edited directly but instead the `geneos set` and `geneos unset` commands should be used to change the configuration parameters.

Note: In a future `cordial` release the configuration file may move to a YAML format for better readability but the JSON format would continue to be supported for backwards compatibility.

### Gateway Configuration Files

#### `gateway.setup.xml`

(Default: Created from template)

When a Gateway instance is created, a simple `gateway.setup.xml` file is created using the template file `${GENEOS_HOME}/gateway/templates/gateway.setup.xml.gotmpl`. This file is used as the setup file for the Gateway and is passed to the Gateway using the `-setup` option. The default template can be changed, per instance, by setting the `config.template` parameter. See the `config.rebuild` and `config.template` parameters below for more details.

#### `instance.setup.xml`

(Default: Created from template)

An `instance.setup.xml` file is created in the Gateway instance directory. This file is built using the template file `${GENEOS_HOME}/gateway/templates/instance.setup.xml.gotmpl` and is "included" by the `gateway.setup.xml` file and contains settings synthesised from the instance configuration. This file is always updated when using the `geneos set`, `geneos unset` and `geneos rebuild` commands, regardless of the `config.rebuild` parameter. If you add environment variables to the instance configuration then they are added to the `instance.setup.xml` file as variables with names of the form `_ENV_VAR_NAME` and string values. For example, if you set an environment variable of `FOO=BAR` then a variable named `_ENV_FOO` with value `BAR` is created in the `instance.setup.xml` file. Any variables you add to the instance configuration using `geneos set gateway GATEWAY -v type:name=value` are also added to the `instance.setup.xml` file as variables with their specified names, types and values. For example, if you set a variable of `NAME=VALUE` with type `string` then a variable named `NAME` with value `VALUE` and type `string` is created in the `instance.setup.xml` file. This behaviour can be customised by setting the `config.template` parameter to use a different template that does not include these variables or includes them in a different way.

### Instance Parameters

For general instance parameters, applicable to all component types, please see the documentation for the `geneos set` command, i.e. `geneos help set`.

The parameters described below are specific to the Gateway component.

* `gatewayname` (Default: Instance Name)

  The Gateway's name can be different to the instance name. This is used in the default templates, under the Operating Environment created in `instance.setup.xml`

* `setup` (Default: `${config:home}/gateway.setup.xml`)

  The Gateway setup file. If this is set to `none` or an empty string then no `-setup` option is passed on the command line. This allows for Centralised Config with Gateway Hub or Obcerv.

* `licdhost` (Default: `localhost`)
* `licdport` (Default: `7041`)
* `licdsecure` (Default: `false`)

  These three parameters control the connection to the license daemon. If `licdsecure` is set to `true` then the Gateway uses TLS to connect to the license daemon.

  When a new Gateway instance is created, `licdsecure` is set to `true` if TLS is enabled for the Gateway. If this is not correct for your environment then you should change it, either on the command like used to create the instance, i.e. `geneos add ... licdsecure=false` or `geneos deploy ... licdsecure=false`, or using `geneos set` later on.

* `usekeyfile` (Default: Depends on the version of the Gateway)

  Whether to use the key file for authentication with the license daemon. If `true` then the `keyfile` parameter is used to specify the path to the key file. If `false` then the `keyfile` parameter is ignored and not passed to the Gateway. The default depends on the version of the Gateway. For versions prior to 5.10.0 the default is `false`. For versions 5.10.0 and later the default is `true` and a keyfile is automatically created for the instance if it does not already exist.

* `keyfile` (Default: `${config:home}/gateway.aes` depending on version)
* `prevkeyfile` (Default: Empty)

  The file paths for the key file and previous key file used for AES256 encryption of secrets in the Gateway configuration files. These are only used if `usekeyfile` is `true`.

  If shared include files contain AES256 encrypted secrets then the same key file should be used across all Gateway instances that use those include files. This is because the key file contains the encryption key and if different key files are used then the secrets cannot be decrypted.

   The `prevkeyfile` is used when rotating keys. When a new key file is generated the old key file should be moved to a safe location and its path set in `prevkeyfile`. This allows the Gateway to decrypt secrets encrypted with the old key file while it is being rotated.

### Optional Parameters

* `insecureport` (Default: Unset)

  An optional additional port to listen on for non-TLS connections. This allows the Gateway to support both TLS and non-TLS connections at the same time.

* `snapshot.username` (Default: Unset)
* `snapshot.password` (Default: Unset)

  Optional username and password for the `snapshot` command to use when connecting to the Gateway REST API. These are used as defaults for the `snapshot` command and can be overridden on the command line by the `--user`/`-u` option. If these parameters are not set and no credentials can be found in the credentials file then the user will be prompted for a username and password when running the `snapshot` command.

### Configuration Parameters for `geneos rebuild`

When creating a new Gateway instance two XML setup files are created.

An `instance.setup.xml` include file is created and contains settings synthesised from the instance configuration. It is always updated when using the `geneos rebuild` command. This file is rebuilt regardless of the `config.rebuild` parameter.

A default `gateway.setup.xml` file is also created from the template(s) installed in the `gateway/templates` directory. By default this file is only created once but can be re-created using the `rebuild` command with the `-F` option if required. In turn this can also be protected against by setting the Gateway configuration setting `config.rebuild` to `never`.

* `config.rebuild` (Default: `initial`)

  The `rebuild` parameter controls how the instance responds to the `geneos rebuild` command. See below for more details.

* `config.template` (Default: `gateway.setup.xml.gotmpl`)

  The `template` parameter controls which template file is used to build the gateway setup file when `geneos rebuild` is run.

* `includes` (Default: Empty)

  A list of include files to be used when building the Gateway setup file from templates.

  Gateways support the setting of include files for use in templated configurations. These are set similarly to the `-e` parameters:

  ```text
  geneos set gateway example2 -i 100:/path/to/include
  ```

  The setting value is `priority:path` and path can be a relative or absolute path or a URL. In the case of a URL the source is NOT downloaded but instead the URL is written as-is in the template output.

### Centralised Configuration

To use Centralised Configuration with either Gateway Hub or Obcerv, you should set the following parameters appropriately. They do not have defaults.

Also, you should set `setup` to either an empty value or the literal `none` to trigger Centralised Configuration on the Gateway.

* `gateway-hub` or `obcerv`

  One of these two parameters should be set to the URL of the Centralised Configuration store. Setting both is not a valid configuration.

* `app-key`

  To authenticate with an application key, set this parameter to the file path. Note that the application key file should be generated with the AES keyfile used by the Gateway **and** updated if the key file is changed.

* `kerberos-principal` and `kerberos-keytab`

  To authenticate using Kerberos, set these parameters as documented in the Gateway Installation Guide.
## Usage

```text
geneos gateway
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos home`

<sub>[Up: geneos](#geneos)</sub>

Output a directory path for use in shell expansion like `cd $(geneos home mygateway)`.

Without arguments, the output will be the root of the Geneos installation, if defined, or an empty string if not. In the latter case a shell running `cd` would interpret this as go to your home directory.

With only a TYPE and no instance NAME the output is the directory root directory of that TYPE, e.g. `${GENEOS_HOME}/gateway`

Otherwise, if the first NAME argument results in a match to an instance then the output is it's working directory. If no instance matches the first NAME argument then the Geneos root directory is output as if no other options were given.

For obvious reasons this only applies to the local host and the `--host`/`-H` option is ignored. If NAME is given with a host qualifier and this is not `localhost` then this is treated as a failure and the Geneos home directory is returned.

If the resulting path contains whitespace your shell will see this as multiple arguments and a typical `cd` will fail. To avoid this wrap the expansion in double quotes, e.g. `cd "$(geneos home 'Demo Gateway')"`. The best solution is to not use white space in any instance name or directory path above it. (Note: We tried outputting a quoted path but the bash shell ignores these quotes inside `$(...)`)

## Usage

```text
geneos home [TYPE] [NAME]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
cd $(geneos home)
cd $(geneos home gateway example1)
cat $(geneos home gateway example2)/gateway.txt

```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos host`

<sub>[Up: geneos](#geneos)</sub>

The `host` sub-system allows you to manage connections to remote Geneos servers.

The `geneos` program can seamlessly manage Geneos instances transparently and securely by connecting to other Linux systems running Geneos components. You can do all the same things as with other Geneos instance that you have created locally.

This can be as simple at:

```bash
geneos add host myServer2
geneos add gateway Gateway2@myServer2
geneos start -H myServer2
```

Almost any command that accepts names to match instances can handle names like `name@host`.

Commands like `ls` and `ps` will works transparently and merge all instances together, showing you where they are configured to run.

The host name, as opposed to _hostname_, that you use when you add a host to `geneos` does not have to be the same as the underlying _hostname_ of the remote server.

## Adding Hosts

Currently only the SSH protocol is supported.

The recommended way to ensure secure connectivity is to use and agent with SSH keys. If this is not available then you can also use locally stored, unprotected (as in without a passphrase) private keys and finally locally stored encrypted passwords.

Future releases may add support for protected private keys (through the local encrypted storage of passphrases) or for Kerberos (GSS-API) authentication.

Please note that any configuration in your `ssh_config` file(s) is ignored. Also, you must have already added the remote SSH server key to your `known_hosts` file or connections will fail with an error.

When you add a host you must use either a local label (`HOST` in the examples and documentation) and/or an SSH URL that refers to the connection details and the remote location of the Geneos installation. The `HOST` is taken from the label, if given, or else the `HOSTNAME` part of the SSH URL.

The standard SSH URL has been extended to support the Geneos installation directory and for the `add host` command it looks like:

`ssh://[USER@]HOSTNAME[:PORT][/PATH]`

If not set, `USER` defaults to your current username. Similarly `PORT` defaults to 22. `PATH` defaults to the local Geneos installation path. The most basic SSH URL of the form `ssh://hostname` results in access as the current user on the default SSH port and rooted in the same directory as the local set-up. Is the remote directory is empty (dot files are ignored) then the standard file layout is created.

`HOSTNAME` can be an IP address, as can the `HOST` label but this is not recommended.

```bash
geneos host add server1 ssh://myserver.example.com
```

### Prerequisites

There are other prerequisites for remote support:

* Remote hosts must be Linux

* The remote user must be configured to use a `bash` shell or similar. See limitations below.

If you can log in to a remote Linux server using `ssh user@server` and not be prompted for a password or passphrase then you are set to go. It's beyond the scope of this README to explain how to set-up `ssh-agent` or how to create an unprotected private key file, so please take a look online.

### Limitations

The remote connections over SSH mean there are limitations to the features available on remote servers:

* Control over instance processes is done via shell commands and little error checking is done, so it is possible to cause damage and/or processes not to to start or stop as expected.

* All actions are taken as the user given in the SSH URL (which should NEVER be `root`) and so instances that are meant to run as other users cannot be controlled. Files and directories may not be available if the user does not have suitable permissions.


## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos host add`](#geneos-host-add)	 | Add a remote host |
| [`geneos host delete / rm / remove`](#geneos-host-delete)	 | Delete a remote host configuration |
| [`geneos host list / ls`](#geneos-host-list)	 | List hosts, optionally in CSV or JSON format |
| [`geneos host set`](#geneos-host-set)	 | Set host configuration value |
| [`geneos host show`](#geneos-host-show)	 | Show details of remote host configuration |
| [`geneos host unset`](#geneos-host-unset)	 | Unset Host Parameters |

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos host add`

<sub>[Up: geneos host](#geneos-host) | [Top: geneos](#geneos)</sub>

Add a remote host `NAME` for seamless control of your Geneos estate.

One or both of `NAME` or `SSHURL` must be given. `NAME` is used as the default hostname if not `SSHURL` is given and, conversely, the hostname portion of the `SSHURL` is used if no NAME is supplied.

The `SSHURL` extends the standard format by allowing a path to the root directory for the remote Geneos installation in the format:

  ssh://[USER@]HOST[:PORT][/PATH]

Here:

`USER` is the username to be used to connect to the target host. If is not defined, it will default to the current username.

`PORT` is the ssh port used to connect to the target host. If not defined the default is 22.

`HOST` the hostname or IP address of the target host. Required.
  
`PATH` is the root Geneos directory used on the target host. If not defined, it is set to the same as the local Geneos root directory.

## Usage

```text
geneos host add [flags] [NAME] [SSHURL]
```

### Options

```text
  -I, --init              Initialise the remote host directories and component files
  -p, --prompt            Prompt for password
  -P, --password SECRET   Password
  -k, --keyfile KEYFILE   Keyfile for encryption of stored password (default /home/peter/.config/docs/keyfile.aes)
  -i, --privatekey PATH   Private key file
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos host add server1
geneos host add ssh://server2:50122
geneos host add remote1 ssh://server.example.com/opt/geneos

```

## SEE ALSO

* [geneos host](#geneos-host)	 - Remote Host Operations

---

# `geneos host delete`

<sub>[Up: geneos host](#geneos-host) | [Top: geneos](#geneos)</sub>

Delete the local configuration referring to a remote host.

## Usage

```text
geneos host delete [flags] NAME...
```

### Options

```text
  -F, --force             Delete instances without checking if disabled
  -R, --all               Recursively delete all instances on the host before removing the host config
  -S, --stop              Stop all instances on the host before deleting the local entry
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos host](#geneos-host)	 - Remote Host Operations

---

# `geneos host list`

<sub>[Up: geneos host](#geneos-host) | [Top: geneos](#geneos)</sub>

List the matching remote hosts.

## Usage

```text
geneos host list [flags] [NAME...]
```

### Options

```text
  -a, --all               Show all hosts
  -j, --json              Output JSON
  -i, --pretty            Output indented JSON
  -c, --csv               Output CSV
  -t, --toolkit           Output Toolkit formatted CSV
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos host](#geneos-host)	 - Remote Host Operations

---

# `geneos host set`

<sub>[Up: geneos host](#geneos-host) | [Top: geneos](#geneos)</sub>

Set options on remote host configurations.

## Usage

```text
geneos host set [flags] [NAME...] [KEY=VALUE...]
```

### Options

```text
  -p, --prompt            Prompt for password
  -P, --password SECRET   password
  -k, --keyfile KEYFILE   Keyfile
  -i, --privatekey PATH   Private key file
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos host](#geneos-host)	 - Remote Host Operations

---

# `geneos host show`

<sub>[Up: geneos host](#geneos-host) | [Top: geneos](#geneos)</sub>

Show details of remote host configurations. If no names are supplied then all configured hosts are shown.

The output is always unprocessed, and so any values in `expandable` format are left as-is. This protects, for example, SSH passwords from being accidentally shown in clear text.

## Usage

```text
geneos host show [flags] [NAME...]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos host](#geneos-host)	 - Remote Host Operations

---

# `geneos host unset`

<sub>[Up: geneos host](#geneos-host) | [Top: geneos](#geneos)</sub>

The `geneos host unset` command allows you to remove parameters from host configurations. This can be used to remove items like encrypted passwords as well as private key file paths. Like the main `geneos unset` command parameters have to be named using the `--key/-k` command line flag and to remove private key files from the list use `--privatekey/-i PATH`. At this time the paths to private key files must be given exactly as in the configuration and you cannot use wildcards.

## Usage

```text
geneos host unset [flags] [TYPE] [NAME...]
```

### Options

```text
  -k, --key KEY           Unset configuration parameter KEY
                          (Repeat as required)
  -i, --privatekey PATH   Private key file
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos host unset rem2 -i /path/to/id_rsa

```

## SEE ALSO

* [geneos host](#geneos-host)	 - Remote Host Operations

---

# `geneos import`

<sub>[Up: geneos](#geneos)</sub>

Import each `SOURCE` to instances. With the `--common`/`-c` option imports are saved in the component sub-directory with the name passed as an argument. See examples below.

The `SOURCE` can be the path to a local file, a URL or '-' for `STDIN`. `SOURCE` may not be a directory.

If `SOURCE` is a file in the current directory then it must be prefixed with `"./"` to avoid being seen as an instance NAME to search for. Any file path with a directory separator already present does not need this precaution. The program will read from `STDIN` if the `SOURCE` '-' is given but this can only be used once and a destination DEST must be defined.

If `DEST` is given with a `SOURCE` then it must either be a plain file name or a descending relative path. An absolute or ascending path is an error.

Without an explicit `DEST` for the destination file only the base name of the `SOURCE` is used. If `SOURCE` is a URL then the file name for the resource from the remote web server is preferred over the last part of the URL.

If the `--common`/`-c` option is used then a TYPE must also be specified. Each component of TYPE has a base directory. That directory may contain, in addition to instances of that TYPE, a number of other directories that can be used for shared resources. These may be scripts, include files and so on. Using a TYPE `gateway` as an example and using a `--common config` option the destination for `SOURCE` would be `gateway/config`

Future releases may add support for directories and.or unarchiving of `tar.gz`/`zip` and other file archives.

## Usage

```text
geneos import [flags] [TYPE] [NAME...] [DEST=]SOURCE...
```

### Options

```text
  -c, --common SUFFIX     Import files to a component directory named TYPE_SUFFIX
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
# import a gateway setup file from a web server
geneos import gateway example1 https://example.com/myfiles/gateway.setup.xml

# import the "license.txt" file to the licd instance example2 but
# with a filename of geneos.lic
geneos import licd example2 geneos.lic=license.txt

# import the "myscript.sh" file into the scripts directory under the
# netprobe example3's working directory
#
# Note: the file will not be made executable
geneos import netprobe example3 scripts/=myscript.sh

# import the file "netprobe.setup.xml" from the current directory to
# the SAN localhost
# 
# Note the leading "./" to disambiguate the file name from an instance
# to match
geneos import san localhost ./netprobe.setup.xml

# import "common_include" into the gateway_shared directory under the gateway are of the installation directory
geneos import gateway -c shared common_include.xml

```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos incident`

<sub>[Up: geneos](#geneos)</sub>

The `incident` subsystem allows you to manage the creation, update and resolution of incidents via the `tools/ims-gateway` program. You can also query for a list of incidents and their details.

Currently this incident subsystem, via the `ims-gateway` program, supports ServiceNow and ServiceDesk Plus.

## Usage

```text
geneos incident
```

## Commands

| Command | Description |
|-------|-------|
| [`geneos incident query`](#geneos-incident-query)	 | Query IMS |
| [`geneos incident raise`](#geneos-incident-raise)	 | Create or update an incident |

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos incident query`

<sub>[Up: geneos incident](#geneos-incident) | [Top: geneos](#geneos)</sub>

Query for a list of incidents and their details. This command is used to query the `tools/ims-gateway` program for a list of incidents and their details. The command can be used to filter the list of incidents based on various criteria such as status, priority, assignee, etc. The command can also be used to display the details of a specific incident by providing the incident ID.

The command relies on a configuration file, normally locates in `${HOME}/.config/geneos/ims.yaml`, to provide the connection details for the `ims-gateway` program. If the configuration file is not found or is invalid then an error will be returned. You can specify an alternative configuration file using the `--config`/`-C` option.

## Usage

```text
geneos incident query [FLAGS] [flags]
```

### Options

```text
  -c, --config string       config file to use
  -i, --ims string          IMS type, e.g. "snow" or "sdp". If not specified, the default is taken from config file
  -T, --snow-table string   ServiceNow table, defaults to incident
  -R, --snow-raw            turn ServiceNow sys_display off, i.e. return raw values instead of display values
  -Q, --query string        query to use for the specified IMS type, e.g. a ServiceNow encoded query or a ServiceDesk Plus JSON query. If not specified, the default is taken from config file
  -f, --format string       output format: "csv" or "json" (default "csv")
      --allow-root          allow running as root (not recommended)
  -H, --host HOSTNAME       Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string     profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos incident](#geneos-incident)	 - Commands for working with incidents

---

# `geneos incident raise`

<sub>[Up: geneos incident](#geneos-incident) | [Top: geneos](#geneos)</sub>

Raise or update an incident.

Used with the environment variables Geneos sets as part of an Alert or an Action, or defined on the command line as NAME=VALUE parameters, these key/value pairs are processed using the settings in the `${HOME}/.config/geneos/ims.yaml` file to determine the content of the incident to be raised or updated.

## Usage

```text
geneos incident raise [FLAGS] [field=value ...] [flags]
```

### Options

```text
  -c, --config string         config file to use
  -i, --ims string            IMS type, e.g. snow or sdp. default taken from config file
  -p, --profile string        profile to use for field creation
  -t, --snow-table incident   ServiceNow table, typically incident
      --allow-root            allow running as root (not recommended)
  -H, --host HOSTNAME         Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string       profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos incident](#geneos-incident)	 - Commands for working with incidents

---

# `geneos includes`

<sub>[Up: geneos](#geneos)</sub>

Geneos gateway includes are reusable pieces of configuration or code that can be incorporated into various parts of the Geneos system to extend functionality or simplify management.

A number of ready-made includes are available to help streamline common tasks and configurations within the Geneos system.

## Commands

| Command | Description |
| --- | --- |
| `geneos includes delete / remove` | Delete unlinked versions, optionally matching a version pattern |
| `geneos includes install / extract` | Install embedded includes or a versioned include archive and missing configuration templates |
| `geneos includes list` | List available and configured include files |
| `geneos includes update` | Update links to a versioned include set |

All commands can be limited to a configured host with the global `--host`/`-H` option. Use `geneos includes COMMAND --help` for command options and output details.

Run `geneos includes install` without an archive to install the embedded include files and configuration templates. `extract` is an alias for `install`. Use `--update` to replace existing top-level files and links with links to the installed version.

## Release Archives

The archive contains a `v`-prefixed semantic-version directory of managed include files and a separate, unversioned `config/` directory of configuration templates. Install it with `geneos includes install ARCHIVE.tar.gz`. Existing configuration templates are never replaced. Missing top-level include links are created; use `--update` to replace existing paths with links to the installed version.


## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos includes delete / remove`](#geneos-includes-delete)	 | Delete unlinked include versions |
| [`geneos includes install / extract`](#geneos-includes-install)	 | Install embedded includes or includes from an archive |
| [`geneos includes list / ls`](#geneos-includes-list)	 | List all available include files |
| [`geneos includes update`](#geneos-includes-update)	 | Update includes |

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos includes delete`

<sub>[Up: geneos includes](#geneos-includes) | [Top: geneos](#geneos)</sub>

Delete unlinked version directories from `gateway/includes` on the selected host. The `remove` alias has the same behaviour.

With no version pattern, all unlinked semantic-version directories are removed and linked versions are skipped. To select versions, supply one or more shell-style patterns as positional arguments, or one pattern using `--version` (`-V`). Patterns match directory names exactly, including any `v` prefix, and support `*`, `?`, and character classes such as `[12]`. Quote patterns so the shell does not expand them. Supplying positional arguments together with the flag is an error.

Before removing any version, the command checks every symlink directly in the parent `gateway/includes` directory. A link to even one file protects the entire version directory, including every unlinked file within it. Relative and absolute targets, link chains, directory aliases, and dangling references into the version are protected.

Positional patterns are processed in order. If a pattern selects any linked version, none of the versions matching that pattern are deleted, and an error identifies the version and its protecting symlink. Later patterns are still processed, and collected errors are returned after the list is complete. If a link cannot be inspected or resolved safely, that pattern fails without proceeding with deletion.

Links originating elsewhere are not considered: those are manually managed by the user. Configuration templates, non-version directories, and directory symlinks are never deleted. Missing includes directories and patterns that match no versions are harmless no-ops.

```shell
geneos includes delete
geneos includes delete 'v1.30.*'
geneos includes delete 'v1.29.*' 'v1.30.*'
geneos includes delete --version 'v1.31.0-rc.*'
geneos includes remove -V 'v1.29.*'
```

Use the global `--host` (`-H`) option to select the installation destination. Links are checked again immediately before each directory is removed. Deleted directories are not backed up; earlier deletions are not rolled back if a later filesystem operation fails or a new protecting link appears during deletion.

## Usage

```text
geneos includes delete [VERSION_PATTERN...] [flags]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
  -V, --version string    Select include versions using a shell-style pattern
```

## SEE ALSO

* [geneos includes](#geneos-includes)	 - Manage includes

---

# `geneos includes install`

<sub>[Up: geneos includes](#geneos-includes) | [Top: geneos](#geneos)</sub>

Install include files embedded in the `geneos` executable or from a local `.tar.gz` archive to the selected host. With no archive argument, the embedded files and configuration templates are installed. `extract` is an alias for `install` and accepts the same arguments and options.

The archive must contain one directory named with a `v`-prefixed semantic version (for example `v1.31.0` or `v1.31.0-rc.1`), containing `itrs.*.xml` include files. Unversioned `config.*.xml` templates belong in a separate `config/` directory. Links, duplicate files, unexpected entries and unsafe paths are rejected before installation begins.

Managed includes are installed under `gateway/includes/<version>`, including when that directory already exists. Changed files are backed up to timestamped `.bak` files; unchanged files are left in place.

Configuration templates are installed under `gateway/includes/config` only when no path of the same name exists. Local configuration files and existing symlinks are never replaced.

Missing top-level include links are created automatically. Existing files and links are left alone unless `--update` (`-U`) is given, which replaces them with links to the installed version. When an archive is supplied, updates target the version in the archive; otherwise they target the version embedded in the executable.

```shell
geneos includes install
geneos includes install --update
geneos includes install geneos-includes.tar.gz
geneos includes install --update geneos-includes.tar.gz
```

The archive is read locally; the global `--host` (`-H`) option selects the installation destination. Completed filesystem operations are not rolled back if a later operation fails.

## Usage

```text
geneos includes install [ARCHIVE.tar.gz] [flags]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
  -U, --update            Update version links for existing include files
```

## SEE ALSO

* [geneos includes](#geneos-includes)	 - Manage includes

---

# `geneos includes list`

<sub>[Up: geneos includes](#geneos-includes) | [Top: geneos](#geneos)</sub>

List include files found in the selected Geneos installation and include references used by its Gateways. The default view groups shared and versioned includes by file name. Use one of `--include`, `--version`, or `--gateway` to select a view.

The include view reports copies, Gateway usage, status flags, modification time, size, and location. Files with duplicate basenames are shown as separate numbered rows beneath a summary row. `--all` also includes Gateway-local files.

The version view lists the current include links and available semantic-version directories. Use `--all` to list each file inside those directories.

The Gateway view lists includes configured for each Gateway, ordered by priority. The `File` field contains only the include file's basename; nested group names do not change it. Gateway-local includes are omitted unless `--all` is set.

Use `--long` for version, permissions, and owner columns where available. Owner information is not currently collected. The `--format` option accepts `column`, `table`, `csv`, `json`, `pretty`, or `toolkit`; column output uses human-readable sizes, while machine-readable formats use byte counts.

Set `--format json` to output the selected view as a compact array of objects using the long fields regardless of `--long`. Permission values use octal strings such as `"0644"`; table output retains symbolic permissions. Fields use lower-camel-case names. Empty fields are omitted, as is `owner` because owner data is not collected. Counts, priorities, and byte sizes are JSON numbers. Unknown or missing sizes are `-1`; a known empty file has size `0`. Set `--format pretty` for the same JSON output with indentation.

In JSON output, the aggregate `flags` field is replaced by optional boolean fields: `disabled` (`D`), `local` (`L`), `readOnly` (`R`), `shared` (`S`), `remote` (`U`), `missing` (`X`), and `optional` (`o`). A field is included with value `true` only when that flag applies; false fields are omitted. Versioned includes are identified by their non-empty `version` field, so JSON does not repeat the `V` flag.

Flags identify include properties:

* `D` - Disabled in the Gateway setup
* `L` - Gateway-local include
* `o` - Optional include
* `R` - Read-only file
* `S` - Shared include
* `U` - HTTP or HTTPS URL include
* `V` - Versioned include
* `X` - Local file is missing

Remote locations are queried with HTTP HEAD requests for modification time and size unless `--no-remote` is set. Failures to retrieve remote metadata do not prevent listing the configured location.
## Usage

```text
geneos includes list [flags]
```

### Options

```text
  -a, --all               List all include files, including those in versioned directories
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -f, --format string     Specify the output format for the include list. Supported formats are 'column', 'csv', 'json', 'pretty', 'table' and 'toolkit'. (default "column")
  -g, --gateway           List include files by gateway usage
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
  -i, --include           List include files by shared include files
  -l, --long              Long format with extra information
  -n, --no-remote         Exclude remote include files lookups
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
  -v, --version           List include files by version
```

## SEE ALSO

* [geneos includes](#geneos-includes)	 - Manage includes

---

# `geneos includes update`

<sub>[Up: geneos includes](#geneos-includes) | [Top: geneos](#geneos)</sub>

Update the top-level include symlinks to point to files in a versioned include directory on the selected host.

By default, the links are updated to the version embedded in the running `geneos` executable. Use `--version` to select another semantic version already present under `gateway/includes`. If the selected version directory is missing, the command returns an error; the current embedded version is extracted automatically when needed.

Links that already point to the selected version are left unchanged. If a target path already exists with a different link or file, it is removed before creating the new symlink.

Version values must be valid semantic versions. This prevents a version argument from selecting a path outside the includes directory.
## Usage

```text
geneos includes update [flags]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
  -V, --version string    Specify version to update includes to
```

## SEE ALSO

* [geneos includes](#geneos-includes)	 - Manage includes

---

# `geneos init`

<sub>[Up: geneos](#geneos)</sub>

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

## Usage

```text
geneos init [PROFILE] [flags] [DIRECTORY]
```

## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos init templates / template`](#geneos-init-templates)	 | Initialise or overwrite templates |

### Options

```text
  -R, --restore PATH                     Restore from backup file PATH
  -X, --no-install                       Don't install any releases after restore
  -M, --minimal                          use a minimal Netprobe release
  -i, --include PRIORITY:[PATH|URL]      An include file in the format PRIORITY:[PATH|URL]
                                         (Repeat as required, gateway only)
  -L, --licence string                   Licence file location (default "geneos.lic")
  -l, --log                              Follow logs after starting instance(s)
  -F, --force                            Ignore existing directories and files and overwrite
  -n, --name string                      Use name for instances and configurations instead of the hostname
  -C, --signing-bundle string            signing bundle in PEM or PFX/PKCS#12 format.
                                         Use a dash ('-') to be prompted for PEM from console.
                                         PFX/PKCS#12 must be files and are identified by the .pfx or .p12 file extension
      --signing-bundle-password SECRET   Password for PFX/PKCS#12 certificate file.
                                         You will be prompted if required and not supplied as an argument.
      --insecure                         Do not create internal certificates for TLS support
  -A, --archive string                   Directory of releases for installation
  -N, --nexus                            Download from nexus.itrsgroup.com. Requires ITRS internal credentials
  -S, --snapshots                        Download from nexus snapshots. Implies -N
  -V, --version VERSION                  Download matching VERSION, defaults to latest. Doesn't work for EL8 archives. (default "latest")
  -u, --username string                  Username for downloads (password prompted)
  -e, --env NAME=VALUE                   Environment variable for instance start-up in the format NAME=VALUE
                                         (Repeat as required)
      --header NAME=VALUE                HTTP header in the format NAME=VALUE
                                         (Repeat as required)
      --allow-root                       allow running as root (not recommended)
  -G, --config string                    config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME                    Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string                  profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos init

```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos init templates`

<sub>[Up: geneos init](#geneos-init) | [Top: geneos](#geneos)</sub>

The `geneos init templates` command is used to write out the default template files for a Geneos installation. Existing template files will be overwritten. Use this command when you adopt a legacy installation or upgrade the `geneos` binary and want to ensure you have the latest default templates.

The `geneos` binary contains embedded template files that are normally written out during initialization of a new installation so that they can be customised if required.

This command will overwrite any files with the same name but will not delete other template files that may already exist.

Use this command if you get missing template errors using the `geneos add` or `geneos deploy` commands.

## Usage

```text
geneos init templates [TYPE] [flags]
```

### Options

```text
      --allow-root                       allow running as root (not recommended)
  -A, --archive string                   Directory of releases for installation
  -G, --config string                    config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -e, --env NAME=VALUE                   Environment variable for instance start-up in the format NAME=VALUE
                                         (Repeat as required)
  -F, --force                            Ignore existing directories and files and overwrite
      --header NAME=VALUE                HTTP header in the format NAME=VALUE
                                         (Repeat as required)
  -H, --host HOSTNAME                    Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --insecure                         Do not create internal certificates for TLS support
  -l, --log                              Follow logs after starting instance(s)
  -n, --name string                      Use name for instances and configurations instead of the hostname
  -N, --nexus                            Download from nexus.itrsgroup.com. Requires ITRS internal credentials
      --profiles string                  profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
  -C, --signing-bundle string            signing bundle in PEM or PFX/PKCS#12 format.
                                         Use a dash ('-') to be prompted for PEM from console.
                                         PFX/PKCS#12 must be files and are identified by the .pfx or .p12 file extension
      --signing-bundle-password SECRET   Password for PFX/PKCS#12 certificate file.
                                         You will be prompted if required and not supplied as an argument.
  -S, --snapshots                        Download from nexus snapshots. Implies -N
  -u, --username string                  Username for downloads (password prompted)
  -V, --version VERSION                  Download matching VERSION, defaults to latest. Doesn't work for EL8 archives. (default "latest")
```

## SEE ALSO

* [geneos init](#geneos-init)	 - Initialise a Geneos installation

---

# `geneos licd`

<sub>[Up: geneos](#geneos)</sub>

A `licd` component is an instance of a Geneos License Daemon. There can only be one running `licd` per system.

## Licence File

When creating the `licd` instance you should import a valid licence file. This is normally supplied by the ITRS Support team named as `HOSTID.lic`, where `HOSTID` is the output of the `hostid` command on the system where the `licd` instance will be running. In the general case, the License Daemon would normally be installed on the standby servers for Geneos deployments where there are primary/standby Gateway configured. In larger environments, the `licd` instance may be installed on a dedicated server to manage licences for multiple Geneos deployments.

If you are using `geneos init all` for a new server that will host the `licd` instance, use the `--licence`/`-L` option with the path to the file and it will be copied into the instance directory with the correct `geneos.lic` file name. If you create the `licd` instance with either `geneos deploy` or `geneos add` then you should use the `--import`/`-I` option but you must specify the destination file name like this: `geneos deploy ... --import geneos.lic=/path/to/HOSTID.lic` or `geneos add ... --import geneos.lic=/path/to/HOSTID.lic`. The `geneos.lic` file is the only licence file that the `licd` instance will use.

To update the licence file for an instance, use the `geneos import` command similar to the above but without the flag, e.g. `geneos import geneos.lic=/path/to/HOSTID.lic`. The `geneos.lic` file will be replaced with the new file but the `licd` instance will not be restarted automatically, so you must also `geneos restart licd` to complete to update.

## Configuration

The License Daemon instance configuration is stored in the instance configuration file `licd.json`. This file is created with the instance and updated when the `geneos set` and `geneos unset` commands are used to change parameters. The configuration file is stored in the instance working directory, which is shown when running the `geneos list` command. This file should not be edited directly but instead the `geneos set` and `geneos unset` commands should be used to change the configuration parameters.

Note: In a future `cordial` release the configuration file may move to a YAML format for better readability but the JSON format would continue to be supported for backwards compatibility.

### Instance Parameters

For general instance parameters, applicable to all component types, please see the documentation for the `geneos set` command, i.e. `geneos help set`.

The License Daemon component has no parameters other than those applicable to all components.

## Usage

```text
geneos licd
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos list`

<sub>[Up: geneos](#geneos)</sub>

List details of matching instances. This command can also be run using the common alias of `ls`.

As for other commands if a `TYPE` is not given all `TYPE`s are included and if no `NAME` is given all instances for `TYPE` are included. Unless `NAME` is given in the format `NAME@HOST` then instances from all hosts are considered. The host can also be controlled using the `--host`/`-H` global option.

The default output is a table format intended for humans but this can be changed to CSV format using the `--csv`/`-c` flag or JSON with the `--json`/`-j` or `--pretty`/`-i` options, the latter option formatting the output over multiple, indented lines. There is also a `--toolkit/-t` output format which produces CSV formatted data suitable for consumption by a Geneos Toolkit plugin.

In plain, table output, format the Flags column contains:

  * `A` - Auto Start
  * `D` - Disabled
  * `P` - Protected
  * `R` - Running
  * `T` - TLS enabled

In other output formats each flag gets it's own column or field.

## Usage

```text
geneos list [flags] [TYPE] [NAME...]
```

### Options

```text
  -c, --csv               Output CSV
  -j, --json              Output JSON
  -i, --pretty            Output indented JSON
  -t, --toolkit           Output Toolkit formatted CSV
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos login`

<sub>[Up: geneos](#geneos)</sub>

The `login` command will store credentials in your local configuration directory for use with `geneos` and other tools from `cordial`. Passwords are encrypted using a key file which is created if it does not already exist.

If not given `DOMAIN` defaults to `itrsgroup.com`. When credentials are used, the destination is checked against all stored credentials and the longest match is selected.

A common use of stored credentials is for the download and installation of Geneos packages via the `geneos package` subsystem. Credentials are also used by the `geneos snapshot` command and the `dv2email` program.

If no `--username`/`-u` option is given then the user is prompted for one.

If no `--password`/`-p` is given then the user is prompted to enter the password twice and it is only accepted if both instances match. After three failures to match password the program will terminate and not save the credential.

The user's default key file is used unless the `--keyfile`/`-k` is given. The path to the key file used is stored in the credential and so if the key file is moved or overwritten then that credential becomes unusable.

The credentials cannot be used without the original key file and each set of credentials can use a separate key file.

The credentials file itself can be world readable as the security is through the use of a protected key file. Running `geneos.exe` on Windows does not currently protect the key file on creation.

Future releases will support extended credential sets, for example SSH and 2-legged OAuth ClientID/ClientSecret (such as application keys from cloud providers). Another addition may be the automatic encryption of non-password data in credentials.

## Usage

```text
geneos login [flags] [DOMAIN]
```

### Options

```text
  -u, --username string   Username
  -p, --password SECRET   Password
  -k, --keyfile KEYFILE   Key file to use
  -l, --list              List the names of the currently stored credentials
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos logout`

<sub>[Up: geneos](#geneos)</sub>

The `logout` command removes the credentials for the `DOMAIN` given. If no names are set then the default credentials (`itrsgroup.com`) are removed.

If the `-A` options is given then all credentials are removed, but the underlying file is not deleted.

## Usage

```text
geneos logout [flags] [DOMAIN...]
```

### Options

```text
  -A, --all               remove all credentials
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos logs`

<sub>[Up: geneos](#geneos)</sub>

The `logs` command shows the log file contents for matching instances. The default behaviour is to show the last 10 lines of the log file for each matching instance. The order of instances cannot be predicted.

You can control the behaviour with three options. The `--lines`/`-n` option controls how many lines to output per instance at the start of the program. The `--cat`/`-c` options will output the whole log file and any `--lines`/`-n` option is ignored. The `--follow`/`-f` option will show the last 10 lines (unless you ask for more with the `--lines`/`-n` option) and then wait for the log to be updated, just like the standard `tail -f` command except it will work for all matching instances including remote ones. `--cat`/`-c` and `--follow`/`-f` are mutually exclusive.

The `--stderr`/`-E` option controls whether the separate `STDERR` log (if there is one) for each matching instance is also shown along with the main log. If used with the `--nostandard`/`-N` option to suppress normal log files then only error output is shown.

The `--ca`/`-C` option controls the inclusion of Collection Agent logs for Netprobes.

The `--match`/`-g` and `--ignore`/`-v` options will filter lines the output based on a case sensitive search over the whole line. As can be expected `--match`/`-g` behaves somewhat like `grep` and `--ignore`/`-v` like `grep -v`. Case-insensitive filtering is avoided for performance.

Only on `--match`/`-g` or `--ignore`/`-v` is allowed.

Each block of output has a header indicating the details of the instance and the path to the log file. The header is output each time the file being output changes. There is no way to suppress this header.

Future releases may add support for more complex filtering using regular expressions and also multiple filters.

## Usage

```text
geneos logs [flags] [TYPE] [NAME...]
```

### Options

```text
  -f, --follow            Follow file
  -n, --lines int         Lines to tail (default 10)
  -c, --cat               Output whole file
  -E, --console           Show console (stdout/stderr) output files
  -N, --no-normal         Do NOT show normal instance log files
  -X, --no-headers        Do not show log file headers, useful for --match/-g output
  -C, --ca                Include Collection Agent log for Netprobe instances
  -A, --audit             Show only audit log files
  -g, --match string      Match lines with STRING
  -v, --ignore string     Match lines without STRING
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos md-gateway`

<sub>[Up: geneos](#geneos)</sub>

An `md-gateway` component is a Java application managed by `geneos`. It is not downloaded from ITRS Geneos servers; install a local product archive.

The process is started with the JDK bundled in the archive (`jdk/bin/java`). To use a different runtime, set `program`. Additional JVM flags can be set with `java-options`, and heap sizes with `xms` and `xmx` (defaults `512m`).

The listening port used by the application is defined in the YAML configuration. The port shown by `geneos list` is only used for instance uniqueness.

## Package layout

Install the released `.tar.gz` or `.zip` as shipped. A wrapping top-level directory is stripped on install. Contents after install:

```text
lib/md-gateway.jar
plugins/*.jar
jdk/bin/java
config/md-gateway.yaml
config/logback.xml
```

Archive names are recognised without `--override`, including Maven snapshots:

```text
md-gateway-1.0.0.tar.gz
md-gateway-1.0.0-SNAPSHOT-linux-x64.tar.gz
md-gateway-1.0.0-20260826.031023-95-linux-x64.tar.gz
md-gateway-asx-1.0.0-20261030.041812-1-linux-x64.tar.gz
```

Each distinct snapshot timestamp is installed as its own version directory. Use `--override md-gateway:VERSION` only if the filename does not match. These packages are not downloaded from ITRS.

Exchange-specific packages, `md-gateway-EXCHANGE-VERSION...`, install as a `md-gateway` release with the exchange appended to the version, e.g. `packages/md-gateway/1.0.0-20261030.041812-1+asx`, so packages for different exchanges with the same version do not collide. To select one exchange, give the package commands `md-gateway-EXCHANGE` as the type:

```bash
geneos package install md-gateway-asx ./downloads/
geneos package install md-gateway-asx -N -S
geneos package update md-gateway-asx
```

This installs only `md-gateway-asx-*` archives from a directory, downloads the `md-gateway-asx` artifact from Nexus, and updates `active_prod` to the latest `+asx` release. Without an exchange, `md-gateway` keeps its existing behaviour. Instances are still `md-gateway` instances.

The `active_prod` link is shared by all exchanges, so on a host with more than one exchange installed, give each instance its own release with `geneos set md-gateway NAME version=1.0.0-20261030.041812-1+asx` rather than relying on `active_prod`.

Releases before v1.30.0 do not recognise exchange package names. To install one with those releases, rename the archive to put the exchange after the version, e.g. `md-gateway-1.0.0-20261030.041812-1-asx-linux-x64.tar.gz`. It then installs into the same `...+asx` directory, and later releases accept the renamed file too.

```bash
geneos package install md-gateway ./md-gateway-1.0.0-20260826.031023-95-linux-x64.tar.gz
```

Limit installation to a remote host with `-H HOST`. The archive path is local.

## YAML configuration

Each instance uses `md-gateway.yaml` in the instance home directory. If that file is missing when the instance is created, it is copied from the packaged `config/md-gateway.yaml`. Supply a custom file when creating the instance:

```bash
geneos add md-gateway prod1 --import md-gateway.yaml=./prod1.yaml --start
```

Or install the package and create the instance in one step:

```bash
geneos deploy md-gateway prod1 --import md-gateway.yaml=./prod1.yaml ./md-gateway-1.0.0-20260826.031023-95-linux-x64.tar.gz -l
```

To update the YAML later:

```bash
geneos import md-gateway prod1 md-gateway.yaml=./prod1-v2.yaml
geneos restart md-gateway prod1
```

The application does not reload YAML on signal; restart after every change.

Local instances write each audit event as one JSON line to the instance audit log (default `md-gateway-audit.log`). Remote instances write only to any configured global audit destination. Records contain an `action`, a `command_id`, `username`, nested `instance` details, a `result`, and action-specific fields. Filter with `jq` on the `action` field:

```json
{"timestamp":"2026-08-28T02:35:10Z","action":"import","command_id":"<uuid>","username":"jenkins","instance":{"component":"md-gateway","name":"prod1","host":"localhost"},"result":"success","source":"./prod1.yaml","file":"md-gateway.yaml"}
{"timestamp":"2026-08-28T02:40:00Z","action":"start","command_id":"<uuid>","username":"jenkins","instance":{"component":"md-gateway","name":"prod1","host":"localhost"},"result":"success","commandline":"/path/to/java ...","pid":12345}
{"timestamp":"2026-08-28T02:45:00Z","action":"stop","command_id":"<uuid>","username":"jenkins","instance":{"component":"md-gateway","name":"prod1","host":"localhost"},"result":"success","pid":12345}
{"timestamp":"2026-08-28T02:46:00Z","action":"restart","command_id":"<uuid>","username":"jenkins","instance":{"component":"md-gateway","name":"prod1","host":"localhost"},"result":"success","newPid":12346,"oldPid":12345}
{"timestamp":"2026-08-28T03:00:00Z","action":"update","command_id":"<uuid>","username":"jenkins","instance":{"component":"md-gateway","name":"prod1","host":"localhost"},"result":"success","version":"1.0.0","base":"active_prod"}
{"timestamp":"2026-08-28T03:05:00Z","action":"set","command_id":"<uuid>","username":"jenkins","instance":{"component":"md-gateway","name":"prod1","host":"localhost"},"result":"success","values":"[xms=512m xmx=1024m]"}
{"timestamp":"2026-08-28T03:10:00Z","action":"add","command_id":"<uuid>","username":"jenkins","instance":{"component":"md-gateway","name":"prod1","host":"localhost"},"result":"success","port":18000}
```

`geneos logs` follows the Java log (`md-gateway.log`), not the audit file. Configure the audit filename with `audit.file`.

The archive also contains `geneos/md-gateway-rules.xml` for a Geneos Gateway include; that file is not used by this component.

## Configuration

Instance settings are stored in `md-gateway.json` in the instance working directory. Use `geneos set` and `geneos unset` rather than editing that file directly.

### Instance Parameters

For general instance parameters see `geneos help set`. Parameters specific to `md-gateway`:

* `setup` (Default: `${config:home}/md-gateway.yaml`) — path to the YAML config passed to the Java main class.
* `jar` (Default: `lib/md-gateway.jar`) — jar path relative to the installed package directory.
* `main-class` (Default: `com.itrsgroup.mdgateway.Main`)
* `program` (Default: bundled `jdk/bin/java` under the selected package version)
* `logback` (Default: packaged `config/logback.xml`)
* `audit.file` (Default: `md-gateway-audit.log`) — audit log path relative to the instance home directory.
* `xms` / `xmx` (Default: `512m`) — JVM heap sizes.
* `java-options` — extra JVM flags, split on spaces.

## Usage

```text
geneos md-gateway
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos migrate`

<sub>[Up: geneos](#geneos)</sub>

The `migrate` command will move instances from old director structures to current ones (e.g. sans from san/sans to netprobe/sans) and also convert legacy `.rc` format files to JSON and named the old file to end `.rc.orig`. The `revert` command can be used to restore these backup files. If you run a full clean (`geneos clean -F`) then it is likely these backup files will be removed.

The `--executables`/`-X` option instead creates symbolic links in the ${GENEOS_HOME}/bin directory for names that match the original `ctl` scripts pointing back to the `geneos` program.

## Usage

```text
geneos migrate [--executables|-X] | [TYPE] [NAME...]
```

### Options

```text
  -X, --executables       Migrate executables by symlinking to this binary
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos minimal`

<sub>[Up: geneos](#geneos)</sub>

A `minimal` Netprobe is one without a Collection agent and associated plugins.

If you will not be using the Collection Agent then this can save about 300MB per download of the release file as well as substantial disk space for the unarchives binaries.
## Usage

```text
geneos minimal
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos move`

<sub>[Up: geneos](#geneos)</sub>

The `move` command will move instances from `SOURCE` to `DESTINATION`. The instance is stopped before, and restarted after, the instance is moved. It is an error to try to move an instance to one that already exists with the same name.

If `TYPE` is not given than each component type that has a matching instance `SOURCE` will be moved to `DESTINATION`. If `DESTINATION` is given as an @ followed by a `host` then the instance is moved to the host but the name retained. In all cases, `host` means the name of a pre-defined `geneos host` remote or `@localhost`, not the server hostname.

Additional `KEY=VALUE` parameters can be supplied on the command line to override or supplement instance parameters from the source instance(s). This can be used, for example, to override the listening port of the destination instance. This may not result in expected behaviour if there are multiple instances being moved. Other instance parameters, such as environment variables, should be set on the destination with the `geneos set` command - or removed entirely with the `geneos unset` command.

Any instance using a legacy .rc file is migrated to a newer configuration file format during the move.

The instance is stopped before, and restarted after, the instance is moved.
 
It is an error to try to move an instance to one that already exists with the same name.

If the component support `geneos rebuild` then this is run after the move but before the restart. This allows SANs to be updated as expected.

Moving across hosts is fully supported.

## Usage

```text
geneos move [TYPE] SOURCE DESTINATION [KEY=VALUE...]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos netprobe`

<sub>[Up: geneos](#geneos)</sub>

A `netprobe` component is an instance of a [Geneos Netprobe](https://docs.itrsgroup.com/docs/geneos/current/collection/netprobe/introduction/netprobe-overview/index.html), which is the primary agent used to collect data from monitored systems.

While there are other component types that represent specific flavours of Netprobe, e.g. `fa2` for the FIX Analyser Netprobe, `minimal` for the non-Collection Agent enabled Netprobe and so on, all of these kinds can also be configured as a normal `netprobe` component, to make overall management easier. On the other hand, the `san` and `floating` component types, while Netprobes, function differently and have different configuration parameters, so they must be configured using their own types.

## Configuration

The Netprobe instance configuration is stored in the instance configuration file `netprobe.json`. This file is created when the instance is created and is updated when the `geneos set` and `geneos unset` commands are used to change parameters. The configuration file is stored in the instance working directory, which is shown when running the `geneos list` command. This file should not be edited directly but instead the `geneos set` and `geneos unset` commands should be used to change the configuration parameters.

Note: In a future `cordial` release the configuration file may move to a YAML format for better readability but the JSON format would continue to be supported for backwards compatibility.

### Instance Parameters

For general instance parameters, applicable to all component types, please see the documentation for the `geneos set` command, i.e. `geneos help set`.

The parameters described below are specific to the Netprobe component.

* `pkgtype` (Default: `netprobe`)

  The package type for this instance. This is used to determine the kind of Netprobe. To select the `pkgtype` when creating a `netprobe` instance you name the instance with a prefix like this: `fa2:faProbe1` or `minimal:minProbe1`. The `pkgtype` is then set to the prefix, e.g. `fa2` or `minimal`, which are currently the only valid alternatives to `netprobe`. If there is no prefix in the instance name then the `pkgtype` is set to `netprobe`.
  
  It should not be changed after the instance is created.

* `calogfile` (Default: `collection-agent.log`)

  The file name of the Collection Agent log file, relative to the `home` directory or an absolute path. This is only used if the Collection Agent is enabled for the Netprobe. This parameters does not control the actual log file created by the Collection Agent, which is defined in a `logback.xml` file, but is used for the `geneos logs` command to locate and display the Collection Agent logs.

* `listenip` (Default: Not set)

  The IP address to listen on. If not set then the Netprobe listens on all available interfaces.

* `hostname` (Default: `localhost`)

  The `HOSTNAME` environment variable is set to this value when the Netprobe is started. It is required by the Collection Agent when self-monitoring is enabled.

## Usage

```text
geneos netprobe
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos package`

<sub>[Up: geneos](#geneos)</sub>

The `package` sub-system commands help you manage Geneos software releases.

For those commands that use them, `VERSION` options are based on the ITRS Geneos release versioning which is now semantic versioning but older versions will have `GA` and `RA` prefixes:

`[GA]X.Y.Z`

Where X, Y, Z are each ordered in ascending numerical order. If a directory starts `GA` it will be selected over a directory with the same numerical versions. All other directories name formats will result in unexpected behaviour. If multiple installed versions match then the lexically latest match will be used. The chosen match may be much higher than that given on the command line as only installed packages are used in the search.


## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos package install`](#geneos-package-install)	 | Install Geneos releases |
| [`geneos package list / ls`](#geneos-package-list)	 | List packages available for update command |
| [`geneos package uninstall / delete / remove / rm`](#geneos-package-uninstall)	 | Uninstall Geneos releases |
| [`geneos package update / upgrade`](#geneos-package-update)	 | Update the active version of installed Geneos package |

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos package install`

<sub>[Up: geneos package](#geneos-package) | [Top: geneos](#geneos)</sub>

The `install` command will, with no other options, download and unarchive the latest releases of each currently installed Geneos component from the official ITRS download servers.

If given directories or file paths on the command line then the install command will only use these as the sources for installation and will not try to download any release. Any values that are not directory paths have symlinks resolved before the underlying filename is checked for component type and version (see `--override` below). This allows, for example, a fixed symlink to the latest version of a local release.

Archive files, URLs and directories can also be supplied with `--archive`/`-A`. This option can be repeated, and its values are added to any positional sources on the command line.

If a component TYPE is given on the command line then installation is limited to that component type, otherwise all components types matching existing components are installed.

To install all component types use the `--all` flag. Its former `-A` shorthand now selects `--archive`, matching `deploy` and `init`.

Components with package variants, such as the exchange packages of `md-gateway` and `tr-gateway`, also accept `TYPE-VARIANT`, e.g. `md-gateway-asx`. Only archives, files and Nexus artifacts for that variant are then used. See `geneos help md-gateway` for details.

💡 Downloads require an ITRS client login and credentials must be provided.

Download credentials can be from those stored locally by the `geneos login` command or those supplied on the command line with the --`username`|`-u` option, which will prompt for a password. In previous versions of cordial you may have also stored credentials in the `geneos` program's own configuration, which will still be used if available.

The deprecated `--pwfile`/`-P` option can be used to refer to a local file containing the password.

If you have already downloaded the release archives then you can use the `--local`/`-L` option to use local files. If you do not supply a file name or directory on the command line then the command will look in the `packages/downloads` directory under the Geneos installation directory.

Downloads are normally saved in the above directory but this can be disabled with the `--nosave`/`-n` option. This is the default if you install locally from a specific directory or file too.

With the `--update`/`-U` option the command will also update the active versions for base specified by `--base`/`-b` (default of `active_prod`) by stopping any instances that use that base name and starting them again after updating the links. Because links are potentially shared by many instances the install may succeed but the update fail if any instances are protected. To also update protected instances use the `--force`/`-F` flag. Note that all matching instances will be stopped, even those that may not be updated, as it is not wholly predictable what version may be installed and the instances must be stopped beforehand.

The `--force`/`-F` flags implies `--update`.

By default the latest version found will be the one installed, either from the download site or locally. To install a specific version from the use the `--version`/`-V` option with a version in the form `MAJOR.MINOR.PATCH` where omitting `PATCH` will get the latest patch release for `MAJOR.MINOR` and omitting `MINOR.PATCH` will get the latest version in the `MAJOR` set. Versions cannot be selected for remote `el8` archives because of a restriction in indexing releases. Specifying a version with either a local only or with a directory name on the command line will apply the same rules to all matching local files. This flag is ignored if specific archive files are given on the command line.

If you have downloaded a release but have changed the file name from the original then you must use the `--override`/`-O` option to inform the `install` command which component type and release version the archive contains, e.g. `-T gateway:5.12.1`. The command will not validate your option and will simply unarchive the file, if it can, in the directory that would be created for that component and version. If this option is used with either more than on file or a directory on the command line then that is an error.

For internal ITRS users there are the `--nexus`/`-N` and `--snapshot`/`-S` options to download archives from the internal nexus server. The `--snapshot`/`-S` option implies `--nexus`/`-N`. You may need to supply different credentials for these downloads.

Installations can be limited to a specific host with the global `--host`/`-H` option otherwise the installation is done to all configured hosts.

Finally, if you just want to download releases and not install them - so you can put them on a shred drive for example - then you can use the `--download`/`-D` option. This will download the selected releases to the current directory, or if you give a directory on the command line then to that directory. Note that this option makes no sense with a number of other command line options and will error if those are given, e.g. `--local/-L` and so on.

To use a proxy, when direct connectivity from your server may not be available, set the appropriate environment variables as detailed in the Go documentation: <https://pkg.go.dev/net/http#ProxyFromEnvironment>. The values of these variables are the same as for the industry-standard examples you will find on the web.

The default timeout for a single download is 60 seconds. This may be too short for larger downloads or if your network connection is slow. You can change this with the `download.timeout` configuration parameter, e.g. `geneos config set download.timeout=120s` to set it to 120 seconds. The value must be a valid Go duration string, e.g. `30s`, `1m`, `1h30m` and so on.

## Usage

```text
geneos package install [flags] [TYPE] [FILE|URL...]
```

### Options

```text
  -u, --username string       Username for downloads, defaults to configuration value in download.username
  -L, --local                 Install from local files only.
                              * Implied if files or directories are given on command line.
  -n, --nosave                Do not save a local copy of any downloads.
                              * Implied if files or directories are given on command line.
  -D, --download              Download only, to the current directory.
                              Cannot be combined with --nosave
  -U, --update                Update the base directory symlink, will restart unprotected instances
  -F, --force                 Force restart of protected instances, implies --update
  -b, --base string           Override the base active_prod link name (default "active_prod")
  -V, --version string        Download this version, defaults to latest. Doesn't work for EL8 archives.
                              Ignored if local file(s) - not directories - are given to install on command-line (default "latest")
  -O, --override string       Override the TYPE:VERSION for archive files with non-standard names
  -N, --nexus                 Download from nexus.itrsgroup.com. Requires auth.
  -S, --snapshots             Download from nexus snapshots (pre-releases), not releases. Requires -N
      --all                   Install all types available, not just those types already installed
  -A, --archive stringArray   URL or file path to release archive
                              or a directory to search for local release archives.
                              Added to positional sources (repeat as required)
      --header NAME=VALUE     HTTP header in the format NAME=VALUE
                              (Repeat as required)
      --allow-root            allow running as root (not recommended)
  -G, --config string         config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME         Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string       profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos install gateway
geneos install fa2 -V 6.5 -U
geneos install netprobe -b active_dev -U
geneos package install md-gateway-asx -N -S

```

## SEE ALSO

* [geneos package](#geneos-package)	 - Package Operations

---

# `geneos package list`

<sub>[Up: geneos package](#geneos-package) | [Top: geneos](#geneos)</sub>

List the packages for the matching TYPE or all component types if no TYPE is given. The `-H` flags restricts the check to a specific remote host.

All timestamps are displayed in UTC to avoid filesystem confusion between local summer/winter times in some locales.

Versions are listed in descending order for each component type, i.e. `latest` is always the first entry for each component.

## Usage

```text
geneos package list [flags] [TYPE]
```

### Options

```text
  -j, --json              Output JSON
  -i, --pretty            Output indented JSON
  -c, --csv               Output CSV
  -t, --toolkit           Output Toolkit formatted CSV
  -I, --instances         Show instances using each version in toolkit mode, using '#' as a separator
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos package](#geneos-package)	 - Package Operations

---

# `geneos package uninstall`

<sub>[Up: geneos package](#geneos-package) | [Top: geneos](#geneos)</sub>

The `package uninstall` commands removes installed Geneos releases and downloaded archive files. By default all releases that are not used by any enabled or running instance are removed with the exception of the "latest" release for each matching component.

To keep the downloaded archives (in `${GENEOS}/packages/download`) use the `--keep`/`-k` flag, otherwise **all** files in that directory are removed, including the latest release and those that are for other components.

If `TYPE` is given then only releases for that component are removed. Similarly, if `--version VERSION` is given then only that version is removed. `VERSION` must be an exact match and multiple versions or version wildcards are not yet supported.

Note that if `TYPE` is for a component type that uses a different underlying release, such as a `san` which could be a `netprobe` or `fa2` under the hood, you have to remove the main `TYPE`.

To remove releases that are in use by protected instances you must give the `--force`/`-F` flag.

To update releases in use by instances, whether running or not, use the `--update`/`-U` flag. Base links are only updated if this flag is given (and `--force` if any instance using it is marked protected) unless all matching instances are disabled. For each release being removed any running instances will first be stopped and base links will be updated to point to the "latest" version (unless the `--all` flag is used). Any instances stopped will be restarted after all other actions are complete.

If the `-all` flag is passed then all matching releases are removed and all running instances stopped and disabled. This can be used to force a "clean install" of a component or before removal of a Geneos installation on a specific host.

If a host is not selected with the `--host HOST` flags then the uninstall applies to all configured hosts. 

Use `geneos package list` to see which releases are installed.

## Usage

```text
geneos package uninstall [flags] [TYPE] [VERSION]
```

### Options

```text
  -b, --base BASE         Remove the base link BASE for matching TYPEs if no instances reference it.
                          This overrides other options below.
  -V, --version VERSION   Uninstall VERSION
  -A, --all               Uninstall all releases, stopping and disabling running instances
  -k, --keep              Keep cached downloads
  -U, --update            Update base links for instances to latest before restarting and removing
  -F, --force             Force uninstall, stopping protected instances first. Also requires --update
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos uninstall netprobe
geneos uninstall --version 7.4.2

```

## SEE ALSO

* [geneos package](#geneos-package)	 - Package Operations

---

# `geneos package update`

<sub>[Up: geneos package](#geneos-package) | [Top: geneos](#geneos)</sub>

The `package update` command sets the base link for the given component `TYPE`, or all types if not given, to the latest version found in the same package directory.

Use `package list` to see which versions are installed.

For components with package variants, such as `md-gateway`, give `TYPE-VARIANT`, e.g. `md-gateway-asx`, to update to the latest release of that variant only.

💡Note: The `--install`/`-I` option has been deprecated in favour of using the `geneos package install` command directly. To install general releases use the `package install` command or, for more limited uses, you can also use the `--install`/`-I` option to download and install packages that match the options given to `update`. To make things simpler, the `--install`/`-I` option only works with predefined credentials (via `geneos login`) and the official download site, but it will use the version given with the `--version`/`-V` option below.

Alternative versions can be selected via the `--version`/`-V` option or by the first argument after options and component. The base link that is updated defaults to `active_prod` but can be set with `--base`/`-b`.

The `package update` command will create new base links given with the `--base`/`-b` option, so if you maintain multiple base links then check the spelling carefully.

Base links that are in use by protected instance are not updated without the `--force`/`-F` option. Because multiple instances of a component often share the same base link, if any instance is protected then no update is done without `--force`/`-F`.

Otherwise, by default any running instances that use the base link that is being upgraded will be restarted around the update. While not recommended you can prevent this by passing a false value to the `--restart`/`-R` option (`--restart=false`). 

## Usage

```text
geneos package update [flags] [TYPE] [VERSION]
```

### Options

```text
  -V, --version string    Update to this version, defaults to latest
  -b, --base string       Base name for the symlink, defaults to active_prod (default "active_prod")
  -R, --restart           Restart all instances that may have an update applied (default true)
  -F, --force             Will also update and restart protected instances
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos package update gateway -b active_prod
geneos package update gateway -b active_dev -V 5.11
geneos package update
geneos package update netprobe --version 5.13.2
geneos package update md-gateway-asx

```

## SEE ALSO

* [geneos package](#geneos-package)	 - Package Operations

---

# `geneos profile`

<sub>[Up: geneos](#geneos)</sub>

# Profiles

Use a profile to deploy and configure a named group of Geneos instances. A profile can create instances, update existing instances, and build Self-Announcing Netprobe (SAN) managed entities from the components installed on a host.

The profile definition is YAML. The installed `profile` instance stores deployment options. It is not a monitoring process.

## Select the Profile File

On Linux, the normal user file is `~/.config/geneos/profiles.yaml`. If `XDG_CONFIG_HOME` is set, the user configuration directory follows that setting. The system configuration location is `/etc/geneos/profiles.yaml`. The command does not search the current directory for this file.

If no profile file is found, `geneos` uses embedded defaults. During initialisation, it writes the defaults to the user configuration directory if the file does not exist. It does not overwrite an existing file.

To select another file, use the global `--profiles` option:

```bash
geneos --profiles ~/config/profiles.yaml profile list
```

Use the same option when you deploy or rebuild that profile. The installed profile instance stores options, not a copy of the YAML definition or its source path.

The supplied configuration is [internal/profile/profiles.defaults.yaml](../../profile/profiles.defaults.yaml). Use it as an example. Put installation-specific changes in your selected profile file, not in the source file.

## Use a Supplied Profile

### Inspect the Definitions

1. List the profiles in the selected file:

   ```bash
   geneos profile list
   ```

    The list includes each profile's description and its source file. Use `--format` (`-f`) to choose `column` (the default), `table`, `csv`, `json`, `pretty` (indented JSON), or `toolkit` output.

2. Read the help for a named profile:

   ```bash
   geneos profile --help motm
   ```

3. Check its definition in your profile file. In particular, check ports, include paths, Gateway Types, and certificate requirements before deployment.

| Profile | Components and purpose |
| --- | --- |
| `all` | A Gateway and Webserver named for the host, a Netprobe named `localhost`, and a License Daemon named `perm`. The License Daemon uses port `7041`. |
| `demo` | A Gateway named `Demo Gateway` with `options=-demo`, a Netprobe, and a Webserver named `Demo Webserver`. The Gateway has best-practice includes. The Webserver port is `8443`, or `8080` when `--insecure` is selected. |
| `motm` | A SAN for Monitor-Of-The-Monitors (MOTM). It requests host monitoring and creates managed entities for installed Gateways, Netprobes, minimal Netprobes, File Agents, License Daemons, and Webservers. |
| `motm-gateway` | A host-named MOTM Gateway with best-practice includes, and a Netprobe. It is the receiving side of an MOTM installation. |

`all` is a starting example, not an inherited base profile. Profiles do not inherit settings from one another.

### Initialise a Demo Installation

Use `init` for a new installation. You need access to suitable release archives or an ITRS download account. The include files referenced by `demo` must be available at the configured paths.

1. Initialise and deploy the profile. Replace the account and absolute directory path:

   ```bash
   geneos init demo /opt/itrs/geneos --username user@example.com
   ```

   Enter the download password when prompted. Use `--archive /opt/itrs/releases` to use previously downloaded release archives. With `init`, `--minimal` selects the minimal Netprobe package for the supplied profiles.

2. Check the installed instances and their status:

   ```bash
   geneos list
   geneos ps
   ```

3. Check generated configuration and logs before you use the installation for monitoring.

`init` requests start-up for the profile instance. Do not assume that its monitoring processes are running: check `geneos ps` and use `geneos start profile demo` if required. For an existing installation, use `geneos deploy profile NAME` to deploy a profile, then use `geneos start profile NAME` to start its instances. See the [init command reference](#geneos-init) and [deploy command reference](#geneos-deploy) for installation and download options.

### Deploy the MOTM SAN

Use this procedure on an existing Geneos host. The receiving Gateway must have the Types requested by the profile. For TLS, both sides must trust the certificate chain.

1. Check the Gateway address and the Types in the `motm` definition.
2. Deploy the profile with an issued certificate bundle. The bundle must contain the SAN's certificate, private key, and certificate chain:

   ```bash
   geneos deploy profile motm \
     --gateway motm-a.example.com:7038 \
     --gateway motm-b.example.com:7038 \
     --attribute ENVIRONMENT=Production \
    --certs-bundle /opt/itrs/certs/motm-san.pem
   ```

   Repeat `--gateway` and `--attribute` as required. For a password-protected PKCS#12 bundle, supply the password as described in the [deploy command reference](#geneos-deploy). Do not put clear-text passwords in the profile file.

3. Start the instances in the profile:

   ```bash
   geneos start profile motm
   ```

4. Confirm that its managed entities appear on the receiving Gateway.

After you add or remove a monitored component, [rebuild the profile](#apply-changes) to update the managed entities.

## Create a Profile

The top-level `profiles` key maps unique profile names to definitions. Each definition contains metadata and component entries.

Use canonical component names where possible. The profile application handlers support `gateway`, `licd`, `netprobe`, `minimal`, `webserver`, `san`, and `floating`. Registered aliases, such as `gateways` and `netprobes`, are also accepted. Do not assume that every component supported by `geneos` has a profile application handler.

Use a YAML list for component instances. A single mapping is also accepted, as shown by the supplied `demo` and `motm` profiles. The list form makes it easier to add another instance.

1. Add this definition to the `profiles` mapping in your selected file. Do not add a second top-level `profiles` key:

   ```yaml
   profiles:
     application:
       _short: Application monitoring
       _description: |
         Deploy an application Gateway and Netprobe.
       gateway:
         - name: application-gateway
           params:
             - gatewayname=Application Gateway
             - port=7100
       netprobe:
         - name: application-probe
           params:
             - port=7105
   ```

2. Confirm that `geneos` can read the file:

   ```bash
   geneos --profiles ~/config/profiles.yaml profile list
   ```

   This lists definitions. It does not validate include files, ports, Types, or certificates.

3. Deploy the profile into your configured installation:

   ```bash
   geneos --profiles ~/config/profiles.yaml deploy profile application
   ```

4. Inspect the resulting configuration:

   ```bash
   geneos show gateway application-gateway
   geneos show netprobe application-probe
   ```

5. Start the profile's instances when the configuration is correct:

   ```bash
   geneos --profiles ~/config/profiles.yaml start profile application
   ```

Choose ports that are free on the host. The Gateway's internal name, `gatewayname`, can differ from the instance name used in commands.

## Common Configuration

### Profile Metadata

| Key | Meaning |
| --- | --- |
| `_short` | Description shown by `geneos profile list`. |
| `_description` | Markdown description shown by `geneos profile --help NAME`. |

Keys that start with `_` are not treated as component entries. The top-level `version` in the supplied file identifies that configuration example; it does not select the software release to install. Use deployment options for release selection.

### Instance Keys

| Key | Format and use |
| --- | --- |
| `name` | Required, non-empty instance name. Supports expansion. Use a unique name within the component type. A package prefix, such as `minimal:probe1`, selects an alternative package where supported. |
| `env` | List of `NAME=VALUE` strings. Sets the process environment for the instance. |
| `params` | List of `KEY=VALUE` strings. Sets instance configuration parameters, such as `port`, `gatewayname`, or `logfile`. The available parameters depend on the component. |
| `imports` | List of `[DEST=]PATH\|URL` strings. Imports files into the instance directory. `DEST` must be relative to, and below, that directory. Without `DEST`, the source file name is used. |
| `cert-bundle` | Certificate bundle source: a file path or PEM data. Used to import instance certificates. |
| `cert-bundle-password` | Password for a password-protected certificate bundle. Supports cordial secret expansion. |

Example instance entry:

```yaml
netprobe:
  - name: application-probe
    env:
      - JAVA_HOME=/usr/lib/jvm/java-17
    params:
      - port=7105
    imports:
      - application.conf=/opt/itrs/config/application.conf
```

For component parameters, see the [show command reference](#geneos-show) and [set command reference](#geneos-set). Profiles record their name in each applied instance's `config.profile` setting.

## Value Expansion

Expansion replaces `${...}` text when the profile is applied. Use it in instance names, common string values, SAN Gateway lists, SAN Types and attributes, and SAN variable values. Do not assume that every YAML key is expanded in the same context.

| Form | Result |
| --- | --- |
| `${hostname}` | Host name supplied by the profile lookup table. |
| `${env:NAME}` | Environment variable from the `geneos` process. An unset variable produces an empty value. |
| `${option:EXPRESSION}` | Result of a `goval` expression using selected command options and `NAME=VALUE` arguments. |
| `${config:KEY}` | Value from the current configuration. In a SAN `for-each` entry, this is the matched instance's configuration. |
| `${expr:EXPRESSION}` | Result of a `goval` expression using current configuration values. Process environment variables are available through the `env` map. |
| `${select:KEY1:KEY2:DEFAULT}` | First non-empty string configuration value, or the final literal default. Use an empty final field to return an empty string. |
| `${replace:KEY1:KEY2:/PATTERN/TEXT/}` | Regular-expression replacement on the first non-empty string configuration value. The final delimiter must match the other delimiters. A delimiter other than `/` is allowed, but not `:`. |

Examples from the supplied profiles:

```yaml
name: ${option:minimal?"minimal:":""}localhost
params:
  - port=${option:insecure?8080:8443}
```

Examples within a SAN `for-each` entry:

```yaml
name: ${replace:gatewayname:name:/ /_/} Gateway
variables:
  - name: logfile
    type: string
    value: ${config:home}/${config:logfile}
```

The option names supplied from flags are `attribute`, `base`, `env`, `gateway`, `include`, `insecure`, `minimal`, `type`, and `variable`, where the command defines them. Flag defaults are included. This is not a list of flags accepted by every command. In particular, `--minimal` is an `init` option, not a `deploy` option.

Additional command arguments in `NAME=VALUE` form can supply option values. For example, a profile can use `name: ${option:site}` and be deployed with:

```bash
geneos deploy profile application site=application-gateway
```

These arguments are saved with the profile options. They are not automatically applied as parameters to every component. Likewise, use `${option:gateway}` or `${option:attribute}` in a definition when you want those command options to control its SAN configuration.

Use `${option:...}`, not `${flag:...}`. Arbitrary flags are not available to option expressions. In the current code, `licence` is not in the included flag list, although the supplied `all` profile uses `${option:licence}`. Do not rely on `init --licence` alone to populate that import; set an explicit import source in your profile.

External lookup expansion is disabled: `${file:...}` and `${https://...}` do not read external content. This does not prevent `imports` from using a path or URL. Expansion errors can produce empty values; inspect the applied configuration before start-up.

## Gateway Configuration

Gateway entries accept the common keys, plus `includes` and `variables`.

### Includes

`includes` maps a unique integer priority to an include path. To place an include in groups, put the group names before the path and separate each field with `|`:

```yaml
gateway:
  - name: application-gateway
    includes:
      500: Configuration | /opt/itrs/includes/application-config.xml
      1000: Applications | Production | /opt/itrs/includes/application.xml
```

Lower numbers have higher priority. The main setup file normally has priority `1`. Spaces around `|` are removed. Includes with the same group path are placed in those groups by the Gateway template.

An include entry references a file; it does not import that file. Use `imports` if you need to copy a file into the instance directory. Relative paths are interpreted by the Gateway in its runtime setup context, not relative to the profile YAML file. Check the generated setup file and make sure the paths are accessible.

### Variables

`variables` defines Operating Environment variables in the generated Gateway runtime setup file:

```yaml
variables:
  - name: environment
    type: string
    value: Production
  - name: retryCount
    type: integer
    value: "3"
```

Each item has `name`, `type`, and `value`. If `type` is omitted, it defaults to `string`. Use string values in YAML, including quoted numbers and booleans.

The variable types are `string`, `integer`, `double`, `boolean`, `activeTime`, `externalConfigFile`, and `secret`. A `secret` requires an instance AES key file. It is encrypted and stored as a string variable.

### Gateway Name and Licensing

Use `params` for these settings:

```yaml
params:
  - gatewayname=Application Gateway
  - licdhost=licence.example.com
  - licdport=7041
  - licdsecure=true
```

The profile structure also declares `gateway-name`, `licd-host`, `licd-port`, and `licd-secure`. The current application handler does not apply those fields. They are not equivalent to the parameters above in this implementation.

## SAN Configuration

A `san` entry accepts common keys and the following keys:

| Key | Format and use |
| --- | --- |
| `gateways` | List of `HOST:PORT` strings. `${option:gateway}` accepts repeated `--gateway` values. |
| `types` | List of Gateway Type names for the primary managed entity. The Types must exist on the receiving Gateway. |
| `attributes` | List of `NAME=VALUE` strings for the primary managed entity. Later values replace earlier values with the same name. |
| `variables` | List of `name`, `type`, and `value` mappings. Uses the [variable format](#variables) described above. |
| `managed-entities` | List of additional managed entity definitions. Each can be fixed or generated from installed instances. |

The primary entity uses the SAN's configured `sanname`. The default template emits an entity only if it has at least one Type. Attributes and variables alone do not create an entity.

With no configured Gateways, the default template connects to `localhost:7038` for TLS, or `localhost:7039` without TLS. Certificate and TLS settings determine whether connections are secure; the port number alone does not enable TLS.

### Additional Managed Entities

| Key | Format and use |
| --- | --- |
| `name` | Name of the additional managed entity. Keep generated names unique. |
| `for-each` | Component type whose installed local instances supply the entity configuration. Omit it for a fixed entity. |
| `match` | List of instance-name glob patterns. With `for-each`, the default is `*`. An instance that matches any pattern is selected. |
| `ignore` | List of instance-name glob patterns applied after `match`. A matching instance is excluded. |
| `types`, `attributes`, `variables` | Same formats as the primary entity. Under `for-each`, expansion uses each matched instance's configuration. |

Patterns use glob syntax, not regular expressions. For example, `prod-*` selects names that start with `prod-`; `?` matches one character. Define explicit values for fixed entities. Use `for-each` when values must come from installed component configuration.

### Generate Entities for Selected Gateways

Add this SAN to a profile. It creates one additional entity for each Gateway whose name starts with `prod-`, except names that start with `prod-test`:

```yaml
san:
  - name: ${hostname}-application-san
    gateways:
      - ${option:gateway}
    types:
      - Level 1 - Infrastructure
    attributes:
      - ${option:attribute}
    managed-entities:
      - for-each: gateway
        match:
          - prod-*
        ignore:
          - prod-test*
        name: ${config:name} Gateway
        types:
          - Level 2 - Geneos Gateway
        attributes:
          - ${option:attribute}
          - COMPONENT=Gateway
          - GATEWAY=${config:name}
        variables:
          - name: gatewayName
            type: string
            value: ${config:gatewayname}
          - name: logfile
            type: string
            value: ${config:home}/${config:logfile}
```

1. Install the Gateway instances to monitor before you apply this profile.
2. Deploy the profile with `--gateway HOST:PORT` and any required `--attribute` options.
3. Inspect the SAN setup file. Confirm the selected names, Types, and file paths.
4. Start or restart the named profile and check the receiving Gateway.

`for-each` examines installed instances when the profile is applied. It does not detect later installation changes automatically.

## Floating Netprobe Configuration

A `floating` entry accepts common keys and `gateways`. It does not accept the SAN-specific `types`, `attributes`, `variables`, or `managed-entities` keys.

```yaml
floating:
  - name: floating-probe
    gateways:
      - ${option:gateway}
```

Supply repeated `--gateway HOST:PORT` options at deployment. The generated configuration uses the instance's TLS settings. Configure the corresponding Floating Netprobe on the receiving Gateways.

## Apply Changes

A rebuild reads the selected YAML definition again and reuses the options saved with the installed profile instance. It applies the definition to its component instances. It does not download software releases.

1. Edit the profile in the same file used for deployment.
2. Apply the definition again:

   ```bash
   geneos --profiles ~/config/profiles.yaml rebuild profile application
   ```

3. Check the affected instances with `geneos show`. For a SAN, check the generated managed entities.
4. If a Gateway's generated files must change, rebuild that Gateway explicitly:

   ```bash
   geneos rebuild gateway application-gateway --force
   ```

5. Restart the profile's instances if required:

   ```bash
   geneos --profiles ~/config/profiles.yaml restart profile application
   ```

The component's `config.rebuild` setting controls generated files. The normal Gateway setting is `initial`; SAN and Floating Netprobe use `always`. `--force` permits a rebuild for `initial`, but not for `never`. A profile rebuild does not automatically reload or restart its child processes. See the [rebuild command reference](#geneos-rebuild).

To change a saved option, set it on the profile instance before rebuilding. For example:

```bash
geneos set profile motm config.options.gateway=motm-c.example.com:7038
geneos rebuild profile motm
```

Use `geneos show profile motm` to inspect the saved options. Passing new SAN flags to `rebuild` is not a substitute for changing those saved values.

Profile application updates the first matching instance of each component and name, or creates it if absent. On existing instances, it clears selected settings before it applies the new values:

| Component | Settings cleared before application |
| --- | --- |
| Gateway | `licdsecure`, `licdport`, includes, variables, and environment. |
| SAN | Gateways, attributes, Types, variables, environment, and additional managed entities. |
| Floating Netprobe | Gateways and environment. |
| Common-only handlers | No predefined clearing list. |

Thus, an omitted list can remove existing settings. Conversely, removing an arbitrary parameter from `params` does not necessarily remove its old value. Back up configuration before you apply a profile to existing instances. An instance with `protected=true` rejects profile application.

## Limits and Fault Isolation

| Symptom or requirement | Check or action |
| --- | --- |
| Profile is not found | Check its name and the selected `--profiles` file. `profile list` lists available definitions; `geneos list profile` lists installed profile instances. |
| Expanded value is empty | Check the option name, process environment, or matched instance parameter. Option-expression errors can produce empty strings. |
| Include file is not found | Check the path from the Gateway runtime setup context. An `includes` entry does not copy a file. |
| SAN entity is absent | Check that it has Types. For generated entities, also check installed instance names, `match`, and `ignore`. |
| A newly installed component is absent from MOTM | Rebuild the `motm` profile, then use `geneos restart profile motm` if required. |
| Gateway setup file does not change | Check `config.rebuild`; explicitly rebuild the Gateway with `--force` if its setting is `initial`. |
| TLS connection fails | Check the trust chain and TLS settings at both ends. Use an instance certificate bundle from the intended trust chain. |

Current implementation limits:

- Use profile application on the local host. Although deployment accepts host selection, the application handlers and SAN `for-each` processing use `geneos.LOCAL`. Do not rely on remote profile deployment to configure the selected host correctly.
- Component entries are processed without a defined dependency order. Install components before a SAN profile that discovers them, then rebuild that profile after installation changes.
- `deploy --signing-bundle` is used when it initialises a local installation. It is not passed through to profile deployment in an existing installation. Use `--certs-bundle` for issued instance certificates there.
- Application is not transactional. An error can leave some instances updated and others unchanged. Check the result before retrying.
- Removing an instance from the YAML does not delete the installed instance. Renaming an entry can create a new instance and leave the old one in place.
- [`geneos profile extract`](#geneos-profile-extract) and its `update` alias write the embedded defaults to the standard user profiles file. If that file differs, it is backed up as `profiles-YYYYMMDDHHMMSS.yaml` before replacement; if it already matches, no file change is made. A custom file selected with `--profiles` is not overwritten.
## Usage

```text
geneos profile
```

## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos profile extract / update`](#geneos-profile-extract)	 | Extract the embedded profile configuration |
| [`geneos profile list / ls`](#geneos-profile-list)	 | List available profiles |

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos profile extract`

<sub>[Up: geneos profile](#geneos-profile) | [Top: geneos](#geneos)</sub>

Extracts the embedded default profile configuration to the user's `geneos/profiles.yaml` file. If the file already matches the embedded defaults, it is left unchanged. If it differs, the existing file is backed up as `profiles-YYYYMMDDHHMMSS.yaml` before the embedded defaults are written. A missing file is created without a backup.

This command always targets the standard user configuration path, even if another profile file is selected with `--profiles`. See the [profile guide](#geneos-profile) for profile file locations and usage.
## Usage

```text
geneos profile extract
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos profile](#geneos-profile)	 - Profile

---

# `geneos profile list`

<sub>[Up: geneos profile](#geneos-profile) | [Top: geneos](#geneos)</sub>

List available profiles with their descriptions and the configuration file they came from. The default `column` format prints a readable table. Use `--format` (`-f`) to select `column`, `table`, `csv`, `json`, `pretty`, or `toolkit`. `pretty` outputs indented JSON. If no profile file is selected, `From` is `internal defaults`.
## Usage

```text
geneos profile list
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -f, --format string     Specify the output format: column, csv, json, pretty, table or toolkit (default "column")
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos profile](#geneos-profile)	 - Profile

---

# `geneos protect`

<sub>[Up: geneos](#geneos)</sub>

The `protect` command marks matching instances as protected. Various operations that affect the state or availability of an instance will be prevented if it is marked `protected`.

To reverse this you must use the same command with the `-U` flag. There is no `unprotect` command. This is by design.

## Usage

```text
geneos protect [flags] [TYPE] [NAME...]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
  -U, --unprotect         unprotect instances
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos ps`

<sub>[Up: geneos](#geneos)</sub>

The `ps` command, which can also be invoked using the alias `status`, will report details of running instances which match any TYPE and/or NAMES given on the command line. It can also report on open files and sockets.

The `--long/-l` flag also reports a number of other metrics, largely derived from the Linux `/proc` filesystem. Also, as it potentially takes significant time to lookup ports for remote instances these are not shown by default. Use the `--long`/`-l` option to see these.

Use `--files`/`-f` to list open files or `--network`/`-n` to list TCP sockets. These flags select different output views and cannot be used together.

The default output is a table format intended for humans but this can be changed to CSV format using the `--csv`/`-c` flag or JSON with the `--json`/`-j` or `--pretty`/`-i` options, the latter option formatting the output over multiple, indented lines. Use the `--toolkit/-t` flag to report in Geneos Toolkit specific CSV format, which includes headlines and a unique first column to act as the row name.

Any Netprobes that have a running Collection Agent managed process will show a `netprobe/ca` entry along with details of the Collection Agent process.
## Usage

```text
geneos ps [flags] [TYPE] [NAMES...]
```

### Options

```text
  -f, --files             Show open files
  -n, --network           Show TCP sockets
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
  -l, --long              Show more output (remote ports etc.)
  -j, --json              Output JSON
  -i, --pretty            Output indented JSON
  -c, --csv               Output CSV
  -t, --toolkit           Output Toolkit formatted CSV
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos rebuild`

<sub>[Up: geneos](#geneos)</sub>

The `rebuild` command will rebuild all matching instances whose TYPE supports templates for configuration file depending on the `config.rebuild` setting for each instance.

The values for the `config.rebuild` option are: `never`, `initial` and `always`. The default value depends on the TYPE; For Gateways it is `initial` and for SANs and Floating Netprobes it is `always`.

You can force a rebuild for an instance that has the `config.rebuild` set to `initial` by using the `--force`/`-F` option. Instances with a `never` setting are never rebuilt.

To change this use something like `geneos set gateway MyGateway config.rebuild=always`

Instances will not normally update their settings when the configuration file changes, although there are options for both Gateways and Netprobes to do this, so you can trigger a configuration reload with the `--reload`/`-r` option. This will send the appropriate signal to matching instances regardless of the underlying configuration being updated or not.

The templates used for each `TYPE` are stored in the `templates/` directory under each `TYPE` directory, e.g `[GENEOS]/gateway/templates`. If you do not have any templates because you are adopting an existing installation then the program will use internal defaults. Use the `geneos init template` command to write out the current templates based on the built-in ones. Only files with the extension `.gotmpl` are treated as templates and directories (and their contents) are ignored.

The rules for how the files are parsed and made available are those for the Go [template.ParseGlob](https://pkg.go.dev/text/template#ParseGlob) function.

## Usage

```text
geneos rebuild [flags] [TYPE] [NAME...]
```

### Options

```text
  -F, --force             Force rebuild
  -r, --reload            Reload instances after rebuild
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos reload`

<sub>[Up: geneos](#geneos)</sub>

The `reload` command sends a reload signal to all matching instances whose `TYPE` supports them.

## Usage

```text
geneos reload [TYPE] [NAME...]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos reset`

<sub>[Up: geneos](#geneos)</sub>

The `reset` command removes files and directories in the matching instance working directory to a near-default state. The instances are stopped, the directories cleaned and the instance restarted. Because the command can have side-effects, to match all instances you must provide the `all` keyword as `geneos reset` will not match any instances by default.

For instances that are "protected" you must use the `--force`/`-F` option to reset the instance even if it is running.

The files and directories that are removed are based on the component type and the global settings using the `clean` and `purge` pattern lists.

## Usage

```text
geneos reset [flags] [TYPE] [NAME...]
```

### Options

```text
  -F, --force             Force reset (require for proteced instances) even if instance is running
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
# Stop all netprobes and remove all non-essential files from working 
# directories, then restart netprobes
geneos reset netprobe

```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos restart`

<sub>[Up: geneos](#geneos)</sub>

The `restart` command restarts the matching instances.

By default this is identical to running `geneos stop` followed by `geneos start`.

If the `--all`/`-a` option is given then all matching instances are started regardless of whether they were stopped by the command, useful if you are not full aware of the runtime state of instances.

Protected instances will not be restarted unless the `--force`/`-F` option is given.

Normal behaviour is to send, on Linux, a `SIGTERM` to the process and wait for a short period before trying again until the process is no longer running. If this fails to stop the process a `SIGKILL` is sent to terminate the process without further action. If the `--kill`/`-K` option is used then the terminate signal is sent immediately without waiting. Beware that this can leave instance files corrupted or in an indeterminate state.

If the `--log`/`-l` option is given then the logs of all instances that are started are followed until interrupted by the user.

The options `--extras`/`-x` and `--env`/`-e` can be used to add one-off extra command line parameters and environment variables to the start-up of the process. This can be useful when you may need to run a Gateway with an option like `-skip-cache` after rotating key-files, e.g. `geneos restart gateway Example -x -skip-cache`.

## Usage

```text
geneos restart [flags] [TYPE] [NAME...]
```

### Options

```text
  -a, --all               Start all matching instances, not just those already running
  -F, --force             Force restart of protected instances
  -K, --kill              Force stop by sending an immediate SIGKILL
  -p, --port uint16       Restart instance matching port (overrides TYPE and NAME)
  -x, --extras string     Extra args passed to process, split on spaces and quoting ignored
  -e, --env NAME=VALUE    Extra environment variable (Repeat as required)
  -l, --log               Run 'logs -f' after starting instance(s)
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos restore`

<sub>[Up: geneos](#geneos)</sub>

The `restore` command restores one or more instances and/or shared directories from an archive created by `geneos backup`.

Use the `--list/-l` flag to show the high-level contents from the archive that match other arguments given.

Without either a component type or at least one instance name or pattern the command will do nothing. If given a component type without instance names all matching instances of that component type will be restored. Without a component type but with instances names, all matching instances, regardless of component type, will be restored. To restore all instances use the name `all`. Instance names can use wildcard patterns in shell (or "glob") format. Using the `--shared/-s` flag to restore any shared files and directories only applies to the matching component type if it given.

Those backup archives that contain a `tls/` directory will have the contents restored, which may overwrite existing files.

If an instance name is given in the format `DEST=SRC` then instances in the archive called `SRC` will be restored but renamed `DEST`. In this case `SRC` cannot be a wildcard.

Existing instances will not be overwritten, similarly for shared directories.

The command accepts a combination of filenames and instance name patterns, with optional renaming, and distinguishes them by validating the arguments. Any arguments that are not valid instance names (or wildcard or rename patterns) are treated as archive files. In case your archive file matches a valid instance name you should either use an absolute path to the file or a `./` prefix.

To read from STDIN use `-` but then you must also specify the compression type used with the `--decompress/-z` flag. Supported values are `gzip`, `bzip2` and `none`.

When restoring an instance any changes to the paths in the instance configuration will be updated to match the destination host's `geneos` root. No component specific files will be changed, including `gateway.setup.xml` and so on.

## Usage

```text
geneos restore [flags] [TYPE] [[DEST=]NAME...]
```

### Options

```text
  -s, --shared            include shared files
  -z, --decompress TYPE   use decompression TYPE, one of `gzip`, `bzip2` or `none`
                          if not given then the file name is used to guess the type
                          MUST be supplied if the source is stdin (`-`)
  -l, --list              list the contents of the archive(s)
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos restore backup.tgz
geneos restore gateway ABC x.tgz

```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos revert`

<sub>[Up: geneos](#geneos)</sub>

The `revert` command will revert the `.rc.orig` suffixed configuration file for all matching instances.

For any instance that is `protected` this will fail and an error reported.

The original file is never updated and any changes made since the original migration will be lost. The new configuration file will be deleted.

If there is already a configuration file with a `.rc` suffix then the command will remove any `.rc.orig` and new configuration files while leaving the existing file unchanged.

If called with the `--executables`/`-X` option then instead of instance configurations the command will remove any symbolic links from legacy `ctl` command in `${GENEOS_HOME}/bin` that point to the command.

## Usage

```text
geneos revert [--executables|-X] | [TYPE] [NAME...]
```

### Options

```text
  -X, --executables       Revert 'ctl' executables
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos router`

<sub>[Up: geneos](#geneos)</sub>

An `router` component is an instance of the Geneos Netprobe Router.

## Configuration

### Initial Setup

The Netprobe Router is created with a default configuration file based on a template embedded in `geneos`. If you have upgraded `geneos` from an earlier release you can extract this template (and update others) using the `geneeos init templates` command. The default template is derived from the example in the `one-to-one` directory of the Netprobe Router package.

### Instance Parameters

For general instance parameters, applicable to all component types, please see the documentation for the `geneos set` command, i.e. `geneos help set`.

The parameters described below are specific to the Netprobe Router component.

* `port` (Default: `4317`)

  The default port that the Netprobe Router listens on for incoming data streams. This is passed as the environment variable `COLLECTOR_PORT` and is used in the default template. In future releases this may be supported as a list of ports.

* `router-name` (Default: Instance Name)

  The Netprobe Router's name, which can be different to the instance name. This is used in the default templates and passed as an environment variable `APP`.

* `reporter-host` (Default: `localhost`)
* `reporter-port` (Default: `4318`)

  The host and port to which the Netprobe Router connects to for outgoing connections. These are passed as environment variables `REPORTER_HOST` and `REPORTER_PORT` and are used in the default template. In real world configurations, there may be multiple outgoing connections and support for this may be added in future releases of Cordial.

* `hostname` (Default: the OS host name or `localhost`)

  The hostname that the Netprobe Router uses to report self-monitoring metrics if they are enabled in the configuration file. This is passed as an environment variable `HOSTNAME`.

* `xms` (Default: `512m`)
* `xmx` (Default: `1024m`)

  The minimum and maximum heap size for the Router process. These are passed to the JVM as `-Xms` and `-Xmx` options. If the parameters are set with the `-Xms` and `-Xmx` prefixes then the values are passed to the JVM as-is. Otherwise the values are passed with the `-Xms` and `-Xmx` prefixes added.

* `logback` (Default: `logback.xml`)

  The name of the logback configuration file to use. By default the `logback.xml` file packaged with the installed distribution is copied into the instance directory at creation time. This will not be updated when any package is updated. If you want to use a different logback configuration file then you can either edit the `logback.xml` file in the instance directory or set this parameter to point to a different logback configuration file.

* `java-options` (Default: Unset)

  Additional options to pass to the JVM when starting the Router process. These are passed as multiple arguments split on white space, similar to the general `options` parameter. The difference is that the settings for `java-options` are passed to the JVM before the `-jar` option and the Router JAR file, whereas the general `options` parameter is passed after the `-jar` option and the Router JAR file.

* `main-class` (Default: `com.itrsgroup.collection.ca.Main`)

  The main class to run in the Router JAR file. This is passed to the JVM as the argument after the `-jar` option and the Router JAR file. The default value is the main class packaged with the installed distribution. If you want to use a different main class then you can set this parameter to point to a different main class in the Router JAR file.

* `setup` (Default: `router-config.yaml`)

  The path to the Netprobe Router configuration file. When creating the instance you can either import this file (using the `--import`/`-i` option) or copy it into the instance directory after creation. The default file is based on a template embedded in `geneos` which is derived from the example in the `one-to-one` directory of the Netprobe Router package. For example:

  ```bash
  geneos add router myrouter --import router-config.yaml=./myrouter.yaml --start
  ```

* `plugins` (Default: `plugins` in package directory)

  This parameter value is used in the default template as the value for the `pluginsDirectory` property in the Netprobe Router configuration file. It is passed as the environment variable `PLUGIN_DIR`. Note the singular for of `PLUGIN` for the environment variable, which is deliberate and based on Collection Agent configurations.


## Usage

```text
geneos router
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos san`

<sub>[Up: geneos](#geneos)</sub>

A `san` component is an instance of a [Geneos Netprobe](https://docs.itrsgroup.com/docs/geneos/current/collection/netprobe/introduction/netprobe-overview/index.html), which is a configuration of the primary agent used to collect data from monitored systems, running in a Self-Announcing mode.

Other component types that represent specific flavours of Netprobe, e.g. `fa2` for the FIX Analyser Netprobe, `minimal` for the non-Collection Agent enabled Netprobe etc., can also be configured as a `san` component. Do this when adding a `san` component by prefixing the instance name with the package type, e.g. `fa2:faSAN1` or `minimal:minSAN1`. The `pkgtype` (see below) is then set to the prefix, e.g. `fa2` or `minimal`, which are currently the only valid alternatives to `netprobe`, the default.

## Configuration

The SAN instance configuration is stored in the instance configuration file `san.json`. This file is created when the instance is created and is updated when the `geneos set` and `geneos unset` commands are used to change parameters. The configuration file is stored in the instance working directory. This file should not be edited directly but instead the `geneos set` and `geneos unset` commands should be used to change the configuration parameters.

Note: In a future `cordial` release the configuration file may move to a YAML format for better readability but the JSON format would continue to be supported for backwards compatibility.

### Instance Parameters

For general instance parameters, applicable to all component types, please see the documentation for the `geneos set` command, i.e. `geneos help set`.

The parameters described below are specific to the SAN component.

* `pkgtype` (Default: `netprobe`)

  The package type for this instance. This is used to determine the kind of Netprobe. To select the `pkgtype` when creating a `san` instance you name the instance with a prefix like this: `fa2:faSAN1` or `minimal:minSAN1`. The `pkgtype` is then set to the prefix, e.g. `fa2` or `minimal`, which are currently the only valid alternatives to `netprobe`, which is the default.
  
  It should not be changed after the instance is created.

* `setup` (Default: `${config:home}/san.setup.xml`)

  The SAN setup file. This is generated from the template file defined in `config.template` when `geneos rebuild` is run (see below). It can be edited manually but any changes will be lost when `geneos rebuild` is run again. Either set `config.rebuild` to `never` or change the template file or use the `geneos set` command to change configuration parameters that are used in the template file.

* `calogfile` (Default: `collection_agent.log`)

  The file name of the Collection Agent log file, relative to the `home` directory or an absolute path. This is only used if the Collection Agent is enabled for the Netprobe. This parameters does not control the actual log file created by the Collection Agent, which is defined in a `logback.xml` file, but is used for the `geneos logs` command to locate and display the Collection Agent logs.

* `listenip` (Default: `none`)

  The IP address for the SAN process to listen on. Normally a SAN does not listen for incoming connections and so the default is `none` to stop the process from opening a listening port. It may be necessary to enable the listening port, for example to enable an API endpoint, but then the `listenip` could be set to `localhost` or a specific IP address to limit which interfaces the SAN listens on.

* `hostname` (Default: `localhost`)

  The `HOSTNAME` environment variable is set to this value when the SAN is started. It is required by the Collection Agent when self-monitoring is enabled.

### Configuration Parameters for `geneos rebuild`

When creating a new SAN a new setup file (`netprobe.setup.xml`) is created in the instance directory and the `setup` parameter is set to point to this file.

The following parameters control how, and if, this file is rebuilt automatically and which template to use in this case:

* `config.rebuild` (Default: `always`)

  The `rebuild` parameter controls how the instance responds to the `geneos rebuild` command. See below for more details.

* `config.template` (Default: `san.setup.xml.gotmpl`)

  The `template` parameter controls which template file is used to build the SAN setup file when `geneos rebuild` is run.

The following structured parameters are used in the default `san.setup.xml.gotmpl` template and so can be set to control the content of the generated setup file. If you change the template then these parameters may not be used.

* `gateways` (Default: Empty)

  The list of Gateway instances that this SAN will attempt to connect to. When setting these values the format is `HOSTNAME:PORT`. If the SAN is configured to support TLS then the connection to the Gateway(s) will be set to use if any of the following are true:

  * A `tls.secure` parameter is set to `true`.
  * A `tls.ca-bundle` is specified.
  * A `tls.verify` parameter is set to `true`.
  * A `certchain` is specified.
  
* `attributes` (Default: Empty)

  Attributes are set using `--attribute`/`-a` with a value in the form `NAME=VALUE`. Note that the `NAME` is case-sensitive, unlike basic parameters. To remove an attribute use `geneos unset -a NAME`.

* `types` (Default: Empty)

  Types are set using `--type`/`-t` and are just the `NAME` of the type. The `NAME` is case-sensitive. To remove a type use `geneos unset -t NAME`.

* `variables` (Default: Empty)

  Geneos User Variables are set using `--variable`/`-v` and have the format `[TYPE]:NAME=VALUE`, where `TYPE` in this case is the type of content the variable stores. The supported variable `TYPEs` are: (`string`, `integer`, `double`, `boolean`, `activeTime`, `externalConfigFile` and `secret`). These `TYPE` names are case sensitive and so, for example, `String` is not a valid variable `TYPE`. Other types may be supported in the future. Variable `NAME` must be unique and setting a variable with the name of an existing one will overwrite not just the `VALUE` but also the `TYPE`.

  The `secret` type is specific to cordial and is used to store secrets in an encoded form. If you do not set a `VALUE` then you will be prompted for one, if you are running `geneos` from a terminal. If the `VALUE` is already encoded in expandable for, e.g. from `geneos aes password` then that will be used without change. Any other `VALUE` will be encoded using your user keyfile and later converted, for Gateway templates, to a Geneos style `stdAESPassword` using the Gateway keyfile. The `secret` type is not valid for other component types.

  Note that in all cases `NAME` is case-sensitive, unlike basic parameters. To remove a variable use `geneos unset -v NAME`.

## Usage

```text
geneos san
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos set`

<sub>[Up: geneos](#geneos)</sub>

The `set` command lets you set one or more configuration parameters for matching instances.

There are two kinds of parameters, basic `NAME=VALUE` pairs, and structured parameters which may include lists and other kinds of values, and are passed to templates for processing. Structured parameters include environment variables, include files and Gateway connection details for Self-Announcing and Floating Netprobes. The parameters that are supported depend on the component TYPE and the templates used to build the instance configuration. Parameters that are common to all component types are described in the "Common Parameters" section below, and component-specific parameters are described in the component help, e.g. `geneos help gateway` for Gateway instances.

## Basic Parameters

Note that `geneos set` will overwrite existing parameters without warning. Setting a basic parameter to an empty string will effectively remove its value. This behaviour may change in a future release and should not be relied on. To remove a parameter use the `geneos unset` command.

The `geneos set` command supports basic parameters given as `NAME=VALUE` pairs on the command line as well as options for structured or repeatable keys. Each basic parameter uses a case-insensitive `NAME` as the key in the instance configuration. You can also use `+=` or `+` to append values to an existing basic parameter (which will also create a parameter if it does not exist), e.g. `options+="-extra option"`. If the value starts with a dash it is assumed to be a new command line parameter and is appended with a space, otherwise it is appended as-is, but you are responsible for ensuring the resulting parameter is correctly formatted, e.g. paths having ":" separators.

Parameter valules can be encoded so that secrets do not appear in plain text in configuration files. Use the `--secure`/`-s` option with a parameter name and optional plaintext value. If no value is given then you will be prompted to enter the secret. The plaintext is encoded using a keyfile, either provided with the `--keyfile`/`-k` option or, for components that support it, the keyfile referenced in the instance configuration. Otherwise the user's default keyfile is used (or created) as needed. The encoded value is the same as produced by `geneos aes password`.

Environment variables can be set using the `--env`/`-e` option, which can be repeated as required, and the argument to the option should be in the format `ENV=VALUE`. In this case the `ENV` is case-sensitive. An environment variable `ENV` will be set or updated for all matching instances under the configuration key `env`. These environment variables are used to construct the start-up environment of the instance. Environments can be added to any component TYPE.

## Structured Parameters

Environment variables can be encoded so that secrets do not appear in plain text in configuration files. Use the `--secureenv`/`-E` option with a variable name and optional plaintext value. If no value is given then you will be prompted to enter the secret. The encoding process is the same as for `--secure`/`-s` above.

Include files (only for Gateways) can be set using the `--include`/`-i` option, which can be repeated. The value must me in the form `PRIORITY:PATH|URL` where priority is a number between 1 and 65534 and the PATH is either an absolute file path or relative to the Gateway instance directory. Shared include files are normally stored in a `../../shared/` directory. Alternatively, a URL can be used to refer to a read-only remote include file. As each include file must have a different priority in the Geneos Gateway configuration file, this is the value that should be used as the unique key for updating include files.

Include file parameters are passed to templates (see `geneos help gateway`) and the template may or may not add additional values to the include file section. Templates are fully configurable and may not use these values at all.

For Self-Announcing and Floating Netprobes you can add or update Gateway connection details with the `--gateway`/`-g` option. These are given in the form `HOSTNAME:PORT`. The `HOSTNAME` can also be an IP address and is not the same as the `geneos host` command labels for remote hosts being managed, but the actual network accessible hostname or IP that the Gateway is listening on. This option can also be repeated as necessary and is applied to the instance configuration through templates, see `geneos rebuild`. To remove a Gateway connection use the `geneos unset ` command.

Three more options exist for Self-Announcing Netprobes to set Attributes, Types and Variables respectively. As above these options can be repeated and will update or replace existing parameters and to remove them you should use `geneos unset`. All of these parameters depend on SAN configurations being built using template files and do not have any effect on their own. See `geneos rebuild` for more information. See the Self-Announcing Netprobe documentation for more details, e.g. `geneos help san`.

Future releases may add other specific options and also may offer a more direct way of configuring SANs and Floating Netprobes to connect to Gateway also managed by the same `geneos` program.

## Common Parameters

All components share a common set of parameters which are described in more detail below. Each component may also support type-specific parameters which are documented in the component help, e.g. `geneos help gateway`.

### Required Parameters

Required parameters always have values, using default values if not changed in the configuration file. Note that in some cases the deault is an empty value, which is different to the parameter being unset. For example, if `logdir` is unset then the default is to use the `home` directory of the instance for logs, but if `logdir` is set to an empty value then no log directory is used and the `logfile` parameter is used as an absolute path.

In the examples below, `${GENEOS_HOME}` is the directory of the Geneos installation, which is normally `/opt/itrs/geneos` but can be set during initialisation, and values like `${config:PARAMETER}` are references to another configuration parameters, which are evaluated and replaced in the resulting value.

* `name` (Default: Instance Name)

  The name of the Gateway. This is used in the default templates, under the Operating Environment created in `instance.setup.xml`. It should not be changed.

* `home` (Read Only: `${GENEOS_HOME}/gateway/gateways/${config:name}`)

  This parameter is read-only and is set based on the instance's directory. `${config:name}` is the instance name, not the Gateway name. This allows you to move the instance directory and have the `home` parameter update accordingly. It is used as the working directory for the Gateway process.

* `install` (Default: `${GENEOS_HOME}/packages/gateway`)

  The installation directory for Gateway releases

* `version` (Default: `active_prod`)

  The version of the component in the the `install` directory above. This is normally the name of a symbolic version (the "basename") which is maintained as a link to a real installation version directory. You can create new symbolic version or tie an instance to an exact installed version. See the `geneos package install` and `geneos package update` commands for more details.

* `binary` (Default: `gateway2.linux_64`)

  The component program filename. Should not be changed.

* `program` (Default: `${config:install}/${config:version}/${config:binary}`)

  The full path to the component executable. The items in the default of the form `${config:NAME}` refer other configuration parameters above.

* `libpaths` (Default: `${config:install}/${config:version}/lib64:/usr/lib64`)

  This parameter is combined with any `LD_LIBRARY_PATH` environment variable to create the `LD_LIBRARY_PATH` used when starting the component. The default is the `lib64` directory of the component installation version and the standard system library directory.

* `cpus` (Default: Empty)

  For local Linux instances, a comma separated list of CPU numbers to set the CPU affinity to when starting the component. The value should be a list of decimal values, including ranges. For example, `0-3,5,7-9` would set the affinity to CPUs 0,1,2,3,5,7,8 and 9. If empty then no CPU affinity is set and the component may be scheduled by the kernel onto any available CPU cores.

* `logfile` (Default: `gateway.log`)

  The file name of the component log file, relative to the `home` directory or an absolute path.

* `logdir` (Default: Unset)

  If set, it is used as the directory for the log file above. If not set (the default) then the `home` directory of the instance is used.

* `port` (Default: First available from `7038-7039,7100-`)

  The default port to listen on. The actual default is selected from the first available port in the range defined in `gateway.ports` in the program settings. If TLS is enabled, which is the default, then the base port is 7038 and 7039 is not selected. If TLS is not enabled then the base port is 7039. If you have multiple Gateways running on the same server then the `geneos add` and `geneos deploy` commands, amongst others, will automatically select the next available port in the range.

  The port range is defined in the top-level configuration as `gateway.ports` and defaults to `7038-7039,7100-`. You can change this using `geneos config set TYPE.ports="..."`. See the `geneos config` command for more details.

* `autostart` (Default: `true`)

  Gateway instances are set to be started with the default `geneos start` command. Setting `autostart` to false is different to using `geneos disable` to stop an instance from running. This can be used for instances that only need to be run occasionally or manually, for example a load monitoring Gateway instance. To start a Gateway that has `autostart` set to false you must give both the type and the name to the `geneos start` command, for example `geneos start gateway example2`.

* `protected` (Default: `false`)

  If `true` then the instance is protected from being changed or deleted by the `geneos start`, `geneos stop`, `geneos restart` or `geneos delete` and similar commands. This is useful for critical instances that should not be accidentally modified or removed. When an instance is protected, any attempt to change or delete it using the above commands will result in an error message unless the command is run with the `--force` option.

  This is different to using `geneos disable` to stop an instance from running. This can be used for instances that should not be changed or deleted, for example a production Gateway instance.

* `env` (Default: Empty)

  Environment variables set for the start-up of the Gateway are stored as an array of `NAME=VALUE` pairs. They should be set and unset using `geneos set -e` and `geneos unset -e` respectively to ensure consistency.

### TLS Parameters

* `tls.certificate` (Default: `${config:home}/gateway.pem`)
* `tls.privatekey` (Default: `${config:home}/gateway.key`)
* `tls.verify` (Default: `false`)
* `tls.ca-bundle` (Default: `${GENEOS_HOME}/tls/ca-bundle.pem`)
* `tls.minimumversion` (Default: `1.2`)

These parameters control TLS for the Gateway. TLS is enabled by default with the certificate and private key files in the instance home directory. If `tls.verify` is set to `true` then the Gateway will verify the remote endpoints it connects to, using the trusted roots in `tls.ca-bundle`. If `tls.verify` is set to `true` but the `tls.ca-bundle` file does not exist then the verification chain is set to an appropriate system default, which is seleected from a list of defaults for typical Linux systems.

If `tls.verify` is set to `true` but the `tls.ca-bundle` file does not exist then the verification chain is set to an appropriate system default, which is seleected from a list of defaults for typical Linux systems.

Deprecated parameters for TLS are also supported for backwards compatibility but should not be used in new configurations. If you are upgrading from an older version of `cordial` there is a `geneos tls migrate` command to help you. These deprecated parameters are (top-level pparameters, not under `tls`):

* `certificate`
* `privatekey`
* `certchain`
* `use-chain`

### Optional Parameters

* `options` (Default: Unset)

  A space separated set of additional options to append to the command line of the Gateway. For example, when you create a "demo" environment using `geneos init demo` the Gateway gets a `option` of `-demo`. The contents are split on space before being passed as individual arguments; this means that it is not possible to use arguments containing spaces, such as a file path.

  To pass extra parameters to the Gateway just once please see the `--extra`/`-x` option of the `geneos start`, `geneos restart` and `geneos deploy` commands.

## Usage

```text
geneos set [flags] [TYPE] [NAME...] [KEY=VALUE...]
```

### Options

```text
  -k, --keyfile KEYFILE               keyfile to use for encoding secrets
                                      default is instance configured keyfile,
                                      or user keyfile if not used by the instance type
  -s, --secure NAME[=VALUE]           encode a secret for NAME, prompt if VALUE not supplied, using a keyfile
  -e, --env NAME=VALUE                Environment variable for instance start-up in the format NAME=VALUE
                                      (Repeat as required)
  -E, --secureenv NAME[=VALUE]        encode a secret for env var NAME, prompt if VALUE not supplied, using a keyfile
  -i, --include PRIORITY:[PATH|URL]   An include file in the format PRIORITY:[PATH|URL]
                                      (Repeat as required, gateway only)
  -g, --gateway HOSTNAME:PORT         A gateway connection in the format HOSTNAME:PORT
                                      (Repeat as required, san and floating only)
  -a, --attribute NAME=VALUE          Attribute in the format NAME=VALUE
                                      (Repeat as required, san only)
  -t, --type NAME                     A type NAME
                                      (Repeat as required, san only)
  -v, --variable [TYPE:]NAME=VALUE    A variable in the format [TYPE:]NAME=VALUE
                                      (Repeat as required, san only)
      --allow-root                    allow running as root (not recommended)
  -G, --config string                 config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME                 Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string               profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos set gateway MyGateway licdsecure=false
geneos set infraprobe -e JAVA_HOME=/usr/lib/java8/jre -e TNS_ADMIN=/etc/ora/network/admin
geneos set -s secret netprobe local1
geneos set netprobe cloudapps1 -e SOME_CLIENT_ID=abcde -E SOME_CLIENT_SECRET

```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos show`

<sub>[Up: geneos](#geneos)</sub>

The `show` command shows the configuration for matching instances or show the Geneos configuration used by an instance (depending on options selected below).

By default the configuration used to manage the matching instances is output as a JSON array of objects. Each instance object contains
`instance` metadata and a `configuration` object.

The output can be written to a file using the `--output`/`-o` option.

For instance types that have Geneos configuration files, i.e. Gateways, Self-Announcing and Floating Netprobes, the `--setup`/`-s` option can be used to view these.

For Gateways the `--merge`/`-m` option tries to output a merged configuration using the Gateway `-dump-xml` option, but this is subject to unexpected behaviour when not all remote include files are reachable and so on.

The `show` command can also be used trigger validation of Gateway configuration(s) with the `--validate`/`--V` option. When validating you can pass a Gateway Hooks directory with `--hooks-dir` which can be used to further trigger external processing of the Gateway configuration. You cannot use `--merge` and `--validate` at the same time as during validation the Gateway will internally merge the configuration.

For normal output each instance's underlying configuration is in an object key `configuration`. Only the objects in this `configuration` key are stored in the instance's actual configuration file and this is the root for all parameter names used by other commands, i.e. for a value under `configuration.licdsecure` the parameter you would use for a `geneos set` command is just `licdsecure`. Confusingly there is a `configuration.config` object, used for template support. Other run-time information is shown under the `instance` key and includes the instance name, the host it is configured on, it's type and so on.

By default the interpolated ("expandable" values are expanded) values are shown. The see the underlying value use the `--raw`/`-r` option.

No values that are encrypted are shown decrypted with or without the `--raw`/`-r` option.
## Usage

```text
geneos show [flags] [TYPE] [NAME...]
```

### Options

```text
  -o, --output string     Output file, default stdout
  -r, --raw               Show raw (unexpanded) configuration values
  -s, --setup             Show the instance Geneos configuration file, if any
  -m, --merge             Merge Gateway configurations using the Gateway -dump-xml flag
  -V, --validate          Validate Gateway configurations using the Gateway -validate flag
      --hooks string      Hooks directory
                          (may clash with instance parameters if set for normal execution)
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos snapshot`

<sub>[Up: geneos](#geneos)</sub>

The `snapshot` command fetches one or more dataviews using the Geneos Gateway REST Commands API. The TYPE, if given, must be `gateway`. A Gateway instance name must be given or the use the wildcard `all`. Not providing a Gateway name or `all` will result in no data being returned.

Authentication to the Gateway is through a combination of command line flags and configuration parameters. If the parameters `snapshot.username` or `snapshot.password` are defined for the Gateway (see `geneos help gateway`) then these are used as a default unless overridden on the command line by the `--user`/`-u` option. The user is only prompted for a password if it cannot be located in the configuration or in saved credentials. As the Gateway may be configured to not require authentication, the absence of a username and password is valid and you will not be prompted for a username if one is not found.

The default output is in JSON format as an array of dataviews, where each dataview is in the format defined in the Gateway documentation at

<https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_commands_tr.html#fetch_dataviews>

Flags to select which properties of data items are available: `-V`, `-S`, `-Z`, `-U` for value, severity, snooze and user-assignment respectively. If none is given then the default is to fetch values only.

To output the results in a form suitable for consumption by a TOOLKIT sampler, use the `--format toolkit` option, which will return the first matching dataview in the special CSV format used by the TOOLKIT sampler.

For newer Netprobes that support multiple dataviews from TOOLKIT scripts, use the `--format toolkit-multi` option, which will return all matching dataviews in the special CSV format used by the TOOLKIT sampler. You must set the fixed number of dataviews expected in the TOOLKIT sampler configuration, which should be matched to the number given to the `--limit`/`-l` option.

The toolkit output formats only support cell/headline values and cannot be used to set the severity, snooze or user-assignment properties of the returned dataviews and so the options that control those are ignored.

To help capture diagnostic information the `-x` option can be used to capture matching xpaths without the dataview contents. `-l` can be used to limit the number of dataviews (or xpaths) but the limit is not applied in any defined order.

## Usage

```text
geneos snapshot [flags] [gateway] [NAME] XPATH...
```

### Options

```text
  -V, --value             Request cell values (default true)
  -S, --severity          Request cell severities
  -Z, --snooze            Request cell snooze info
  -U, --userassignment    Request cell user assignment info
  -u, --username string   Username
  -f, --format string     Output format (json, toolkit, toolkit-multi) (default "json")
  -l, --limit int         limit matching items to display. default is unlimited. results unsorted.
  -x, --xpaths            just show matching xpaths
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos sso-agent`

<sub>[Up: geneos](#geneos)</sub>

An `sso-agent` component is an instance of the Geneos SSO Agent to provide Single-Sign On authentication to other Geneos components.

The sso-agent requires Java to be installed and by default looks for `/usr/bin/java`. To use a different Java runtime, set the `JAVA_HOME` environment variable for the instance using `geneos set sso-agent -e JAVA_HOME=/path/to/java` before starting the instance. This can also be set during the `add` and `deploy` commands.

If you already have a stand-alone instance of the SSO Agent and want to bring it under the control the `geneos` program you can do something like this:

```bash
$ geneos deploy sso-agent --import /path/to/existing/sso-agent/conf --user email@example.com
Password:
```

This will download and install the latest SSO Agent install package from the ITRS download site using the credentials `email@example.com` and prompting you for your password. The `--import` option pulls in the existing configuration files (including the keystore/trust store) from the path given. Be careful to not end the path in a `/`.

Once this is done you can start the `sso-agent` instance and it should continue working like the original, but under the management of the `geneos` command.

To deploy a new `sso-agent` instance, use the same command but leave out the `--import ...` arguments. Then you have to edit the `conf/sso-agent.conf` file in the instance directory (`geneos ls` to see the paths or `geneos home sso-agent NAME` to get just the specific path) to set the correct listening port and other mandatory parameters. Use `geneos rebuild sso-agent NAME` to regenerate other files after editing `sso-agent.conf`.

>[!NOTE]
>Even though the commands above tells you it has added an instance with a specific port number, the actual listening port is set in the `conf/sso-agent.conf` file and must be updated by hand. `geneos` will not change the actuual component configuration file automatically.

The sso-agent trust store and key store files (normally the same file) are initialised during deployment and if you have configured TLS then the instance certificate and private key are added to the files as well as a freshly created `ssokey` for the process to sign the authentication tokens.

If you change the `ssl_alias` or any of the passwords for the keystore or truststore, then any command that rebuilds the instance configuration will also rebuild the keystore and truststore files.

As documented in the official SSO Agent documentation, you can use environment variables instead of clear-text passwords in the `conf/sso-agent.conf` file. The `geneos` program will use the `ITRS_SSO_KEYSTORE_PW` and `ITRS_SSO_TRUSTSTORE_PW` environment variables if they are set in the instance configuration when building the keystore and truststore files, respectively. You can set AES256 encoded values in the instance configuration using `geneos set sso-agent NAME --secureenv ITRS_SSO_KEYSTORE_PW` (or `ITRS_SSO_TRUSTSTORE_PW`) and be prompted for the password. In this case you must comment out the `password` fields in the `conf/sso-agent.conf` file. Other supported variables are passed to the sso-agent in it's starting environment, but are not specially interrpreted by `geneos`.

## Usage

```text
geneos sso-agent
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos start`

<sub>[Up: geneos](#geneos)</sub>

The `start` command starts the matching instances unless they are already running (or marked as `disabled`). Use the `restart --all` command to restart all instances regardless of their current state.

The start-up command and environment can be seen using the `geneos command` command.

Any matching instances that are marked as `disabled` are not started. If an instance has a parameter `autostart` set to `false` then this is only started if the name is given explicitly on the command line, e.g. `geneos start gateway Example` but not `geneos start gateway` or `geneos start all`. This allows you to have instances that are not started by default but can be started when needed without having to change the configuration.

With the `--log`/`-l` option the command will follow the logs of all instances started, including the STDERR logs as these are good sources of start-up issues.

The options `--extras`/`-x` and `--env`/`-e` can be used to add one-off extra command line parameters and environment variables to the start-up of the process. This can be useful when you may need to run a Gateway with an option like `-skip-cache` after rotating key-files, e.g. `geneos start gateway Example -x -skip-cache`.

## Usage

```text
geneos start [flags] [TYPE] [NAME...]
```

### Options

```text
  -p, --port uint16       Start instance matching port (overrides TYPE and NAME)
  -x, --extras string     Extra args passed to process, split on spaces and quoting ignored
  -e, --env NAME=VALUE    Extra environment variable (Repeat as required)
  -l, --log               Follow logs after starting instance
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos stop`

<sub>[Up: geneos](#geneos)</sub>

The `stop` command stops the matching instances.

Stop will  show how long each proceess took to stop or kill.

Protected instances will not be restarted unless the `--force`/`-F` option is given.

Normal behaviour is to send, on Linux, a `SIGTERM` to the process and wait for a short period before trying again until the process is no longer running. If this fails to stop the process a SIGKILL is sent to terminate the process without further action. If the `--kill`/`-K` option is used then the terminate signal is sent immediately without waiting. Beware that this can leave instance files corrupted or in an indeterminate state.
## Usage

```text
geneos stop [flags] [TYPE] [NAME...]
```

### Options

```text
  -p, --port uint16       Stop instance matching port (overrides TYPE and NAME)
  -F, --force             Stop protected instances
  -K, --kill              Force immediate stop by sending an immediate SIGKILL
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos tls`

<sub>[Up: geneos](#geneos)</sub>

The `tls` sub-system allows you to manage certificates and associated resources for [Geneos Secure Communications](https://docs.itrsgroup.com/docs/geneos/current/SSL/ssl_ug.html).

You can import and manage your own certificates or create your own certificates with your own certificate authority (also known, incorrectly, as "self-signed" certificates).

Commands allow for initialisation, create and renewal of certificates as well as listing details and copying a certificate chain to all other hosts.

Each instance typically uses the following parameters:

* `tls.certificate` - the path to a certificate file in PEM format
* `tls.privatekey` - the path to a private key file for the certificate above
* `tls.ca-bundle` - the path to a file containing one or more PEM formatted certificates that form a trust chain
* `tls.verify` - a boolean parameter that controls the use of the chain file above

Those components which may offer TLS support on listening ports will do so if the `tls.certificate` and `tls.privatekey` parameters are defined. The contents of the files are not validated and are passed to the undelying Geneos binaries as-is. If the files do not exist or are not valid then the component will fail to start.

Those components that act as clients and connect to servers, Geneos or otherwise, will validate the connection based on the `tls.ca-bundle` and `tls.verify` settings. If these are not set or the file does not exist then the connection is still established using TLS but is not verified to be using a trusted certificate.

Please refer to the component documentation (e.g. `geneos help gateway`) for more details on how TLS is used for that component.

Java based components, such as `webserver` and `sso-agent` will also support custom paths to keystore/truststore files, which are also supported by `geneos`. See the documentation for those components for more details.


## Commands

| Command / Aliases | Description |
|-------|-------|
| [`geneos tls create`](#geneos-tls-create)	 | Create standalone certificates and keys |
| [`geneos tls export`](#geneos-tls-export)	 | Export certificate bundle including private key |
| [`geneos tls import`](#geneos-tls-import)	 | Import certificates (signer if no instances specified) |
| [`geneos tls info`](#geneos-tls-info)	 | Info about certificates and keys |
| [`geneos tls init`](#geneos-tls-init)	 | Initialise the TLS environment |
| [`geneos tls list / ls`](#geneos-tls-list)	 | List certificates |
| [`geneos tls migrate`](#geneos-tls-migrate)	 | Migrate certificates to the new TLS layout |
| [`geneos tls new`](#geneos-tls-new)	 | Create instance certificates and keys |
| [`geneos tls renew`](#geneos-tls-renew)	 | Renew instance certificates |
| [`geneos tls sync`](#geneos-tls-sync)	 | Sync remote hosts certificate chain files |

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos tls create`

<sub>[Up: geneos tls](#geneos-tls) | [Top: geneos](#geneos)</sub>

# `geneos tls create`

Creates a new TLS certificate bundle including a private key and trust chain. This can either be an instance bundle, suitable for use with remote Geneos components, for example Windows Netprobes, or a signing certificate bundle for use on other systems where `geneos` will be used.

The command can also be used to just create a private key without a certificate using the `--private-key`/`-K` option. In this case the file will be created with the name "private-key.pem" instead of being derived from the Common Name.

In both cases the key type can be specified using the `--key-type`/`-k` option. If not specified then the key type of the signing certificate will be used or a default if that cannot be determined. The default key type is currently ECDH but this may change in future releases.

By default the command will create a file containing a private key, certificate and trust chain in the current working directory using the local hostname as the Common Name (`CN`). You can specify a different Common Name using the `--cname`/`-c` option. If you want to include spaces please remember to quote the name. The resulting file has spaces in the Common Name replaced with dashes.

To change to output directory use the `--dest`/`-D` option. The name of the file is always derived from the common name, with spaces replaced by dashes, and a `.pem` extension. The only exception to this is when using an output destination of `-` which will write the output to standard output instead of a file.

The default expiry period is 365 days (from one minute in the past - this is to allow some overlap with system clock issues) unless you use the `--expiry`/`-E` option. This option is ignored when creating a signing certificate with the `--signer`/`-s` option, as signing certificates always have a fixed validity period of 5 years.

If the Geneos installation has not been initialised or the TLS sub-system not been initialised then an error will be returned. With the `--force`/`-F` flags a new root and intermediate certificate are created along with private keys in your user configuration directory unless they already exist. Beware that initialising the TLS subsystem will also result in any new instances you create having certificates automatically created and various options set to trigger the use of TLS by default.

You can add Subject Alternative Names (SANs) to the certificate using the `--san-dns`/`-s`, `--san-ip`/`-i`, `--san-email`/`-e` and `--san-url`/`-u` options respectively. These options can be repeated as required to add multiple SANs of each type. SANs are ignored when creating a signing certificate with the `--signer`/`-s` option. If no DNS SANs are specified then the Common Name will be added as a DNS SAN automatically.

If a certificate already exists for the specified Common Name then an error will be returned unless you use the `--force`/`-F` option to overwrite it.

## Usage

```text
geneos tls create [flags]
```

### Options

```text
  -o, --output directory   Write to directory. Files have the common name as the basename.
                           For bundles use a dash '-' for stdout. (default ".")
  -c, --cname CNAME        Common Name for certificate. CNAME defaults to hostname except.
                           Ignored for --signing.
  -S, --signing CNAME      Create a new signing certificate bundle using CNAME for the Common Name.
                           If written to a directory the file will be called
                           "geneos-signing-certificate-(CNAME).pem".
                           Requires a root certificate ("rootCA.pem") and private key ("rootCA.key")
                           to be present in the user configuration directory.
  -E, --expiry days        Certificate expiry duration in days. Ignored for --signing (default 365)
  -F, --force              Runs "tls init" (but do not replace existing root and signing)
                           and overwrite any existing file in the 'out' directory
  -K, --private-key        Create a private key without a certificate. Ignored for --signing.
  -k, --key-type KeyType   Key type to use for new certificate and/or private key (rsa, ecdsa, ed25519
                           or ecdh). Ignored for --signing. Defaults to the same type as
                           the signing certificate key or a default type if that cannot be determined.
  -s, --san-dns VALUE      Subject-Alternative-Name DNS Name (repeat as required).
                           Ignored for --signing.
  -i, --san-ip VALUE       Subject-Alternative-Name IP Address (repeat as required).
                           Ignored for --signing.
  -e, --san-email VALUE    Subject-Alternative-Name Email Address (repeat as required).
                           Ignored for --signing.
  -u, --san-url VALUE      Subject-Alternative-Name URL (repeat as required).
                           Ignored for --signing.
      --allow-root         allow running as root (not recommended)
  -G, --config string      config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME      Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string    profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos tls](#geneos-tls)	 - TLS Certificate Operations

---

# `geneos tls export`

<sub>[Up: geneos tls](#geneos-tls) | [Top: geneos](#geneos)</sub>

# `geneos tls export`

Exports either the local signer private key, certificate and root certificate suitable for importing into another cordial `geneos` installation or, if you specify either component TYPE or instance NAMEs then the private key, instance certificate and other certificates up to and including the trust root for matching instances is output instead.

The default output is to the console but you can specify a file destination using the `--dest`/`-D` option.

## Usage

```text
geneos tls export [flags] [TYPE] [NAME...]
```

### Options

```text
  -o, --output FILE       Output to FILE, default to stdout
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos tls export --out file.pem
geneos tls export gateway mygateway

```

## SEE ALSO

* [geneos tls](#geneos-tls)	 - TLS Certificate Operations

---

# `geneos tls import`

<sub>[Up: geneos tls](#geneos-tls) | [Top: geneos](#geneos)</sub>

# `geneos tls import`

Import a certificate bundle. If no instance TYPE of NAMEs are provided then this is assumed to be a signer bundle containing the private key, signer certificate and root certificate in PEM format. If you specify a TYPE or NAMEs then the input is treated as an instance bundle instead, which must contains the private key, instance certificate and any intermediate certificates up to and including the trust root.

Note: Because a file name like `bundle.pem` can also be valid instance name you must use a `./bundle.pem` style path to force it to be treated as a file name when importing an instance bundle.

In all cases the bundle must contain a verifiable trust chain from the most specific certificate (instance or signer) to the trust root. If this is not the case then the bundle will be rejected. The private key must also match the first certificate.

The input can be in either PEM or PFX/PKCS#12 format. If the input is in PFX/PKCS#12 format then you must provide the password to decrypt it using the `--password` option or you will be prompted for it.

Any trust root certificates in the bundle will be added to the local trust store if they are not already present.

## Usage

```text
geneos tls import [flags] [TYPE] [NAME...] [PATH]
```

### Options

```text
  -p, --password SECRET   Plaintext password for PFX/PKCS#12 file decryption.
                          You will be prompted if not supplied as an argument.
                          PFX/PKCS#12 files are identified by the .pfx or .p12
                          file extension and only supported for instance bundles
  -k, --key file          Private key file for certificate, PEM format only
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
# Import a certificate bundle for the netprobe instance on localhost from a PEM file
geneos tls import netprobe localhost /path/to/file.pem
# Import a signer bundle from a PEM file
geneos tls import /path/to/file.pem

```

## SEE ALSO

* [geneos tls](#geneos-tls)	 - TLS Certificate Operations

---

# `geneos tls info`

<sub>[Up: geneos tls](#geneos-tls) | [Top: geneos](#geneos)</sub>

This command displays information about TLS certificates, either in local files or from remote endpoints. It can show details about the certificate chain, including verification status, expiration dates, and more.

You can pass multiple files or endpoints to check, and the output can be formatted in different ways, including a human-readable table or a format suitable for further processing.

Once flags have been parsed, any remaining arguments are treated as file paths to check. These files can be in PEM, PFX/PKCS12 or Java Keystore format. If the file is a PFX/PKCS12 or Java Keystore, you can specify the password using the `--password` flag, otherwise you will be prompted to enter it for each file. If you specify a password as a flag then all password protected files must have the same password.

The `--connect`/`-c` flag allows you to specify remote endpoints to connect to and retrieve TLS certificate information. The endpoint should be a URL or in the format `host:port` (`port` defaults to 443 if not given). You can specify multiple endpoints by using the flag multiple times. You can also give the path to a file contraining a list of endpoints (in the same format for `--connect`), one per line, using the `--connect-file` flag.

The output format is controlled by the `--format`/`-f` flag. The default is `column`, which produces a human-readable table. The `toolkit` format produces output in a format suitable for ingestion by a Geneos Toolkit sampler, with headlines for total counts and rows of data for each certificate. Other formats are `csv` and `table`. `csv` doesn't include the headline counts, and `table` is similar to `column` but with borders around the table.

By default all certificates found will be listed. To only show leaf certificates, use the `--leaf`/`-L` flag, which will process but not display CA certificates.

By default a short format is used, which includes only the most important information. If you use the `--long`/`-l` flag, additional details about each certificate will be included in the output.

For certificate verification you can specify additional root certificate files using `--roots`/`-r`. These should be in PEM format. The system's default root certificates and the geneos `ca-bundle.pem` files will also be used for verification, and these cannot be disabled, if they exist.

A certificate is considered verified if it can be chained to a trusted root certificate and is not expired. The hostname is *NOT* used for verifiation. The output includes a "Verified" column which indicates whether the certificate is verified. If a private key is present in a file and it matches a certificate then the PrivateKeyMatched colum will show `true`. In general this will only apply to local files.

The total number of certificates processed, the number of verified certificates, the number of expired certificates, and the number of certificates expiring within 30 days are included in the output headlines when using the `toolkit` format.

Note: This command will work without an initialised Geneos environment, and there is also a `geneos.exe` Windows build which can be used on Windows systems without installing the full Geneos package. However, if you run this command in an environment where the `geneos` program is available, it will use the `ca-bundle.pem` file from the Geneos installation for certificate verification if it exists.

## Toolkit Sampler Example

Here is an example of how you might use this command in a Geneos Toolkit sampler:

Set the `Sampler Script` to:

```
/usr/local/bin/geneos tls info --format toolkit --connect-file ./cert-paths.txt --long --leaf-only
```

And then under the (lower) Advanced tab, in the Script box, set Contents to a list of endpoints (one per line) and Filename to `./cert-paths.txt`

Attach this sampler to a Managed Entity where the `geneos` program is installed (edit the path above to suit) and it should display one row for each endpoint that can be connected. Endpoints that cannot be contacted will show an error in the `CommonName` column.

## Usage

```text
geneos tls info [PATH...] [flags]
```

### Options

```text
  -L, --leaf-only             Only output leaf certificates (i.e. skip CA certificates and any
                              certificate without a matching private key in any file)
  -l, --long                  Output long format (more columns)
  -f, --format string         Output format (column, table, csv, toolkit) (default "column")
  -p, --password SECRET       Password for PFX/PKCS#12 file(s), if needed. Defaults to prompting for each file. Use -p "" to specify empty password.
  -c, --connect connect       Connect to a URL or HOST[:PORT] to get TLS certificates. Can be specified multiple times for multiple instances.
      --connect-file string   Path to file containing list of URLs or HOST[:PORT] to connect to (one per line) to get TLS certificates.
  -r, --roots roots           Path to additional root certificates to use when verifying TLS connections. Can be specified multiple times for multiple files. If not specified, the system root CAs and the Geneos ca-bundle files will be used (and are always included).
      --allow-root            allow running as root (not recommended)
  -G, --config string         config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME         Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string       profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos tls](#geneos-tls)	 - TLS Certificate Operations

---

# `geneos tls init`

<sub>[Up: geneos tls](#geneos-tls) | [Top: geneos](#geneos)</sub>

# `geneos tls init`

Initialise the TLS environment by creating a self-signed root certificate to act as a CA and a signing certificate signed by the root. Any instances will have certificates created for them but configurations will not be rebuilt.

To recreate the root and signing certificates and keys use the `--force`/`-F` option. If what you want to do is to update the signer certificate then use the `geneos tls renew --signer` command instead.

All certificates are created with corresponding private keys. These keys are in ECDH format by default but this can be overridden using the `--keytype`/`-K` option which supports the following formats: "ecdh", "ecdsa", "ed25529" and "rsa". Once set for the root CA, all subsequent certificates will be created using the same key type. You should avoid "ed25519" as this is not supported by normal web browsers and will make it impossible to use the ORB diagnostic interfaces of Geneos.
## Usage

```text
geneos tls init
```

### Options

```text
  -K, --keytype KeyType   Key type for root. One of ecdh, ecdsa, ed25519 or rsa
  -F, --force             Overwrite any existing root and signing certificates
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos tls](#geneos-tls)	 - TLS Certificate Operations

---

# `geneos tls list`

<sub>[Up: geneos tls](#geneos-tls) | [Top: geneos](#geneos)</sub>

List certificates and their details. The root and signing certs are only shown if the `--all`/`-a` flag is given. A list with more details can be seen with the `--long`/`-l` flag, otherwise options are the same as for the main ls command.

Certificates for each instance are validated and the "Valid" column or field contains the boolean result. An instance certificate is valid if **all** the following are true:

* The file path from the instance `certificate` parameter is readable, is in PEM format and can be parsed as an x509 certificate
* The file path from the instance `privatekey` parameter is readable, is in PEM format and matches the certificate above
* The certificate is inside it's validity period
* The certificate can be verified against the trust chain from one of:
    * the file path from the instance `certchain` parameter
    * the installation global `tls/geneos-chain.pem` file
    * the system certificate pool
* The certificate also conforms to other checks done by <https://pkg.go.dev/crypto/x509#Certificate.Verify>

The Common Name (`CN`) and the Subject Alternative Names (`SAN`) values in the certificate are not otherwise checked as Geneos does not use these.

## Usage

```text
geneos tls list [flags] [TYPE] [NAME...]
```

### Options

```text
  -a, --all               Show all certs, including root and signing certs
  -l, --long              Long output
  -j, --json              Output JSON
  -i, --pretty            Output indented JSON
  -c, --csv               Output CSV
  -t, --toolkit           Output Toolkit formatted CSV
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos tls](#geneos-tls)	 - TLS Certificate Operations

---

# `geneos tls migrate`

<sub>[Up: geneos tls](#geneos-tls) | [Top: geneos](#geneos)</sub>

# `geneos tls migrate`

Migrate existing Geneos TLS certificates and related files to the updated layout and naming conventions.

IMPORTANT: This command modifies existing files and configurations and does not have a "revert" or roll-back option. It is strongly recommended to back up your existing TLS files before running this command. Use the `geneos backup` command to create a backup.

Without a component TYPE or instance NAMEs specified, this command will migrate all Geneos components with TLS configurations.

The following steps are performed during migration:

1. Existing instance certificate and chain files are loaded and merged into a single PEM file. Any trust roots are written to the global `cs-bundle.pem` file. NO VALIDATION of certificates or private keys is performed during this step, unlike other TLS commands.

2. For Java based `sso-adapter` and `web-server` components, Java KeyStores (JKS) and TrustStores are used as the primary source of certificates and private keys. These are then converted to PEM format for consistency with other components. Existing PEM files are ignored for these components.

3. The instance parameters are updates to use the new names and the old parameters are deleted.

The instance parameters are updated as follows:

| Old Parameter         | New Parameter              | Des                                                                     |
| --------------------- | -------------------------- | ----------------------------------------------------------------------- |
| `certificate`         | `tls.certificate`         | Path to the instance certificate file                                   |
| `privatekey`          | `tls.privatekey`          | Path to the instance private key file                                   |
| `certchain`           | `tls.ca-bundle`           | Path to the CA bundle file, defaults to the global `ca-bundle.pem` file |
| `usechain`            | `tls.verify`              | If set to false then no certificate verification is performed           |
| `truststore`          | `tls.truststore`          | Path to the truststore file (for webserver only)                        |
| `truststore-password` | `tls.truststore-password` | Password for the truststore file (for webserver only)                   |
| n/a                   | `tls.minimumversion`      | Sets the minimum TLS version for connections                            |

Each component type will use the above parameters as appropriate. For example, the `truststore` and `truststore-password` parameters are only applicable to the `webserver` component.

## Usage

```text
geneos tls migrate [TYPE] [NAME...] [flags]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -c, --check             check if instance is already migrated
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos tls](#geneos-tls)	 - TLS Certificate Operations

---

# `geneos tls new`

<sub>[Up: geneos tls](#geneos-tls) | [Top: geneos](#geneos)</sub>

Create a leaf certificate and private key for each matching instance, signed by the local Geneos signing certificate. The command writes the files to the instance's configured TLS paths, updates its TLS parameters, and saves the instance configuration.

The local TLS environment must be initialised and its signing private key must be available. A matching instance that already has a valid certificate is left unchanged; use [`geneos tls renew`](#geneos-tls-renew) to replace an existing certificate.

The certificate expiry defaults to 365 days. Set `--expiry` (`-E`) to change the lifetime.

Examples:

```bash
# Create certificates for one Gateway
geneos tls new gateway mygateway

# Create certificates for all Netprobes with a two-year lifetime
geneos tls new netprobe all --expiry 730
```


## Usage

```text
geneos tls new [TYPE] [NAME...] [flags]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -E, --expiry int        Certificate expiry duration in days (default 365)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos tls](#geneos-tls)	 - TLS Certificate Operations

---

# `geneos tls renew`

<sub>[Up: geneos tls](#geneos-tls) | [Top: geneos](#geneos)</sub>

Renew certificates for matching instances, or renew the local signing certificate with `--signing`. Instance renewal preserves the existing certificate's subject alternative names and defaults to a 365-day lifetime (`--expiry`/`-E`). Instances without an existing certificate are skipped; use [`geneos tls new`](#geneos-tls-new) to create one.

The instance renewal modes are mutually exclusive:

- By default, write the renewed certificate and key to the active paths and update the instance configuration.
- `--prepare` (`-P`) writes the new certificate and key with a `-new` suffix without replacing the active files or restarting the instance. Check that the staged files are present before continuing.
- `--roll` (`-R`) promotes the staged files and keeps the previous active files with a `-old` suffix.
- `--unroll` (`-U`) restores the previous files from the `-old` suffix and leaves the renewed files staged with the `-new` suffix.

`--roll` and `--unroll` replace files but do not restart the instance. The running process continues to use its currently loaded certificate until it is restarted. This lets you stage certificates during active monitoring and schedule activation and restart for a maintenance window.

> [!IMPORTANT]
> Before using staged renewal on an instance that has not been migrated to the current TLS layout, migrate and validate it separately. `geneos tls renew` runs the TLS migration step before applying any renewal mode, and that step can update the instance files and configuration.

Use `--signing` to renew the local signing certificate instead of instance certificates. This requires the local root certificate and private key; `--expiry` does not apply to signing certificates.

### Staged renewal workflow

```bash
# During normal operation, stage the new certificate and key
geneos tls renew gateway mygateway --prepare

# During a maintenance window, activate them and restart the Gateway
geneos tls renew gateway mygateway --roll
geneos restart gateway mygateway

# If validation fails, restore the previous files and restart again
geneos tls renew gateway mygateway --unroll
geneos restart gateway mygateway
```

`--unroll` restores the saved files; restarting afterward makes the Gateway load them. Confirm the Gateway is healthy before ending the maintenance window.

### Renew the signing certificate

```bash
# Renew the local signing certificate
geneos tls renew --signing
```


## Usage

```text
geneos tls renew [TYPE] [NAME...] [flags]
```

### Options

```text
      --signing           Renew the signing certificate instead of instance certificates
  -E, --expiry int        Instance certificate expiry duration in days.
                          (No effect with --signing) (default 365)
  -P, --prepare           Prepare renewal without overwriting existing certificates
  -R, --roll              Roll previously prepared certificates and backup existing ones
  -U, --unroll            Unroll previously rolled certificates to restore backups
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos tls](#geneos-tls)	 - TLS Certificate Operations

---

# `geneos tls sync`

<sub>[Up: geneos tls](#geneos-tls) | [Top: geneos](#geneos)</sub>

# `geneos tls sync`

The `geneos tls sync` command synchronises TLS certificates and keys from the local TLS environment to remote Geneos instances. This is useful when you have initialised or renewed certificates locally and need to distribute them to the instances. It also forces a rebuild of the local `ca-bundle.db` from `ca-bundle.pem` file which is a Java trust store containing the root CA certificate. The command will also remove any invalid or expired trust roots from the `ca-bundle.pem` file before rebuilding the trust store.

## Usage

```text
geneos tls sync [flags]
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos tls](#geneos-tls)	 - TLS Certificate Operations

---

# `geneos tr-gateway`

<sub>[Up: geneos](#geneos)</sub>

A `tr-gateway` component is a Java application managed by `geneos`. It is not downloaded from ITRS Geneos servers; install a local product archive.

The process is started with the JDK bundled in the archive (`jdk/bin/java`). To use a different runtime, set `program`. Additional JVM flags can be set with `java-options`, and heap sizes with `xms` and `xmx` (defaults `512m`).

The listening port used by the application is defined in the YAML configuration. The port shown by `geneos list` is only used for instance uniqueness.

## Package layout

Install the released `.tar.gz` or `.zip` as shipped. A wrapping top-level directory is stripped on install. Contents after install:

```text
lib/tr-gateway.jar
plugins/*.jar
jdk/bin/java
config/tr-gateway.yaml
config/logback.xml
```

Archive names are recognised without `--override`, including Maven snapshots:

```text
tr-gateway-1.0.0.tar.gz
tr-gateway-1.0.0-SNAPSHOT-linux-x64.tar.gz
tr-gateway-1.0.0-20260826.031023-95-linux-x64.tar.gz
tr-gateway-idx-1.0.0-20261030.041812-1-linux-x64.tar.gz
```

Each distinct snapshot timestamp is installed as its own version directory. Use `--override tr-gateway:VERSION` only if the filename does not match. These packages are not downloaded from ITRS.

Exchange-specific packages, `tr-gateway-EXCHANGE-VERSION...`, install as a `tr-gateway` release with the exchange appended to the version, e.g. `packages/tr-gateway/1.0.0-20261030.041812-1+idx`, so packages for different exchanges with the same version do not collide. To select one exchange, give the package commands `tr-gateway-EXCHANGE` as the type:

```bash
geneos package install tr-gateway-idx ./downloads/
geneos package install tr-gateway-idx -N -S
geneos package update tr-gateway-idx
```

This installs only `tr-gateway-idx-*` archives from a directory, downloads the `tr-gateway-idx` artifact from Nexus, and updates `active_prod` to the latest `+idx` release. Without an exchange, `tr-gateway` keeps its existing behaviour. Instances are still `tr-gateway` instances.

The `active_prod` link is shared by all exchanges, so on a host with more than one exchange installed, give each instance its own release with `geneos set tr-gateway NAME version=1.0.0-20261030.041812-1+idx` rather than relying on `active_prod`.

Releases before v1.30.0 do not recognise exchange package names. To install one with those releases, rename the archive to put the exchange after the version, e.g. `tr-gateway-1.0.0-20261030.041812-1-idx-linux-x64.tar.gz`. It then installs into the same `...+idx` directory, and later releases accept the renamed file too.

```bash
geneos package install tr-gateway ./tr-gateway-1.0.0-20260826.031023-95-linux-x64.tar.gz
```

Limit installation to a remote host with `-H HOST`. The archive path is local.

## YAML configuration

Each instance uses `tr-gateway.yaml` in the instance home directory. If that file is missing when the instance is created, it is copied from the packaged `config/tr-gateway.yaml`. Supply a custom file when creating the instance:

```bash
geneos add tr-gateway prod1 --import tr-gateway.yaml=./prod1.yaml --start
```

Or install the package and create the instance in one step:

```bash
geneos deploy tr-gateway prod1 --import tr-gateway.yaml=./prod1.yaml ./tr-gateway-1.0.0-20260826.031023-95-linux-x64.tar.gz -l
```

To update the YAML later:

```bash
geneos import tr-gateway prod1 tr-gateway.yaml=./prod1-v2.yaml
geneos restart tr-gateway prod1
```

The application does not reload YAML on signal; restart after every change.

Local instances write each audit event as one JSON line to the instance audit log (default `tr-gateway-audit.log`). Remote instances write only to any configured global audit destination. Records contain an `action`, a `command_id`, `username`, nested `instance` details, a `result`, and action-specific fields. Filter with `jq` on the `action` field:

```json
{"timestamp":"2026-08-28T02:35:10Z","action":"import","command_id":"<uuid>","username":"jenkins","instance":{"component":"tr-gateway","name":"prod1","host":"localhost"},"result":"success","source":"./prod1.yaml","file":"tr-gateway.yaml"}
{"timestamp":"2026-08-28T02:40:00Z","action":"start","command_id":"<uuid>","username":"jenkins","instance":{"component":"tr-gateway","name":"prod1","host":"localhost"},"result":"success","commandline":"/path/to/java ...","pid":12345}
{"timestamp":"2026-08-28T02:45:00Z","action":"stop","command_id":"<uuid>","username":"jenkins","instance":{"component":"tr-gateway","name":"prod1","host":"localhost"},"result":"success","pid":12345}
{"timestamp":"2026-08-28T02:46:00Z","action":"restart","command_id":"<uuid>","username":"jenkins","instance":{"component":"tr-gateway","name":"prod1","host":"localhost"},"result":"success","newPid":12346,"oldPid":12345}
{"timestamp":"2026-08-28T03:00:00Z","action":"update","command_id":"<uuid>","username":"jenkins","instance":{"component":"tr-gateway","name":"prod1","host":"localhost"},"result":"success","version":"1.0.0","base":"active_prod"}
{"timestamp":"2026-08-28T03:05:00Z","action":"set","command_id":"<uuid>","username":"jenkins","instance":{"component":"tr-gateway","name":"prod1","host":"localhost"},"result":"success","values":"[xms=512m xmx=1024m]"}
{"timestamp":"2026-08-28T03:10:00Z","action":"add","command_id":"<uuid>","username":"jenkins","instance":{"component":"tr-gateway","name":"prod1","host":"localhost"},"result":"success","port":19000}
```

`geneos logs` follows the Java log (`tr-gateway.log`), not the audit file. Configure the audit filename with `audit.file`.

## Configuration

Instance settings are stored in `tr-gateway.json` in the instance working directory. Use `geneos set` and `geneos unset` rather than editing that file directly.

### Instance Parameters

For general instance parameters see `geneos help set`. Parameters specific to `tr-gateway`:

* `setup` (Default: `${config:home}/tr-gateway.yaml`) — path to the YAML config passed to the Java main class.
* `jar` (Default: `lib/tr-gateway.jar`) — jar path relative to the installed package directory.
* `main-class` (Default: `com.itrsgroup.trgateway.Main`)
* `program` (Default: bundled `jdk/bin/java` under the selected package version)
* `logback` (Default: packaged `config/logback.xml`)
* `audit.file` (Default: `tr-gateway-audit.log`) — audit log path relative to the instance home directory.
* `xms` / `xmx` (Default: `512m`) — JVM heap sizes.
* `java-options` — extra JVM flags, split on spaces.

## Usage

```text
geneos tr-gateway
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos unset`

<sub>[Up: geneos](#geneos)</sub>

The `unset` command removes configuration parameters from matching instances. This command is `unset` rather than `remove` as that is reserved as an alias for the `delete` command.

Unlike the `geneos set` command, where parameters are in the form of KEY=VALUE, there is no way to distinguish a KEY to remove and a possible instance name, so you must use one or more `--key`/`-k` options to unset each simple parameter.

WARNING: Be careful removing keys that are necessary for instances to be manageable. Some keys, if removed, will require manual intervention to remove or fix the old configuration and recreate the instance.

You can also unset values for structured parameters. For `--include`/`-i` options the parameter key is the `PRIORITY` of the include file set while for the other options it is the `NAME`. Note that for structured parameters the `NAME` is case-sensitive. See the usage flags for more details.

## Usage

```text
geneos unset [flags] [TYPE] [NAME...]
```

### Options

```text
  -k, --key KEY            Unset configuration parameter KEY
                           (Repeat as required)
  -e, --env NAME           Remove an environment variable NAME
                           (Repeat as required)
  -i, --include PRIORITY   Remove an include file with PRIORITY
                           (Repeat as required, gateways only)
  -g, --gateway NAME       Remove the gateway NAME
                           (Repeat as required, san and floating only)
  -a, --attribute NAME     Remove the attribute NAME
                           (Repeat as required, san only)
  -t, --type NAME          Remove the type NAME
                           (Repeat as required, san only)
  -v, --variable NAME      Remove the variable NAME
                           (Repeat as required, san only)
  -M, --entity NAME        Remove a managed entity NAME
                           (Repeat as required, san only)
      --allow-root         allow running as root (not recommended)
  -G, --config string      config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME      Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string    profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## Examples

```bash
geneos unset gateway GW1 -k aesfile
geneos unset san -g Gateway1

```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos version`

<sub>[Up: geneos](#geneos)</sub>

The `version` command shows program version details.

With the `--debug`/`-d` option it will also output build information which may be useful when reporting issues or to check for suspected problems in build dependencies.

## Usage

```text
geneos version
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
  -t, --toolkit           toolkit formatted CSV output
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments

---

# `geneos webserver`

<sub>[Up: geneos](#geneos)</sub>

A `webserver` is an instance of a Web Dashboard Server.

A new `webserver` instance is created using local package configuration files, therefore the same package version must be installed locally as on any remote host.

`geneos` will manage TLS certificates for the webserver instance. When you create a new instance, assuming your Geneos installation has TLS initialised, a new certificate and private key will be created as for other component types but these will also then be added to the Java keystore referenced in the `config/security.properties` file. If you replace or renew the instance certificate or private key you should then use the `geneos rebuild webserver NAME` command to rebuild the keystore file. If you change the `security.properties` file you may need to manually delete the (old) keystore file before running `rebuild`.

If you use `geneos deploy webserver` you can also provide an externally generated certificate bundle without having to initialise TLS for other components.

## Usage

```text
geneos webserver
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](#geneos)	 - Take control of your Geneos environments
