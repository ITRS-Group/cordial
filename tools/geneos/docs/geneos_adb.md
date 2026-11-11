# `geneos adb`

# Active Dashboard Files

Use `geneos adb` to inspect and update exported Active Console dashboard files. These files normally have an `.adb` extension. They are ZIP archives that contain dashboard definitions, resources, and optional startup data.

The commands operate on files. They do not connect to a Gateway or change a running dashboard. Import the result into Active Console to use it.

## Select Input Files

```text
geneos adb info [flags] [PATH...]
geneos adb clean [flags] PATH...
geneos adb edit [flags] PATH...
```

`dashboard` and `dashboards` are aliases for `adb`; `edit` is an alias for `clean`. The commands accept multiple file paths. Quote paths that contain spaces. The shell can expand a file pattern; the commands do not search directories recursively.

```bash
geneos adb info /opt/itrs/dashboards/application.adb
geneos adb info /opt/itrs/dashboards/*.adb
```

No Geneos installation directory is required for these subcommands. `clean` requires at least one input file. With no input, `info` produces no report.

## Inspect a Dashboard

`info` reads the `.Activedashboard` entries inside each archive. It does not change the input file.

| Option | Default | Use |
| --- | --- | --- |
| `--show-xpaths`, `-X` | `false` | List sorted, distinct XPaths only. |
| `--show-urls` | `false` | List sorted, distinct URL destinations only, without link wrappers or source labels. |
| `--show-all` | `false` | List both XPaths and URLs in separate blocks. |

Without a list option, the report lists Gateway, Probe, Managed Entity, Attribute, Sampler, Type, and Dataview names found in supported data-item references. Each heading shows the number of distinct names. Each name has an occurrence count. For Attributes, the report also shows distinct values and their counts. A lack of name for an element is shown as `<no name>`. An empty Type name appears as `<empty>`, while the lack of a Type on a Sampler is not recorded.

1. Get a summary of one dashboard:

   ```bash
   geneos adb info /opt/itrs/dashboards/application.adb
   ```

2. Show its data-item paths:

   ```bash
   geneos adb info --show-xpaths /opt/itrs/dashboards/application.adb
   ```

3. Use the names and paths to select the changes needed before you run `clean`.

Counts and XPath/URL lists accumulate across files in one invocation. For independent reports, run `info` once per file. The report is not a full inventory of all dashboard objects.

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

Use the `edit` alias when making targeted changes without the default cleanup: it preserves Gateway, Probe, Managed Entity, and Type values and `.dbo` startup data unless explicit options request changes.

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

After each file is processed, `clean` and `edit` report the number of updated XPath and URL links, which XPath components changed, and any modifier aliases or startup-data entries removed. A file with no changes is reported as such.

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

The generated command references are [adb](../../docs/geneos_adb.md), [info](../../docs/geneos_adb_info.md), and [clean](../../docs/geneos_adb_clean.md). The current implementation is in [cmd/adbcmd]().

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

| Command / Aliases | Description |
|-------|-------|
| [`geneos adb clean / edit`](geneos_adb_clean.md)	 | Clean or edit ADB file(s) |
| [`geneos adb info`](geneos_adb_info.md)	 | Info about ADB file(s) |

### Options

```text
      --allow-root[=true|false]   allow running as root (not recommended)
  -G, --config string             config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME             Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string           profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
