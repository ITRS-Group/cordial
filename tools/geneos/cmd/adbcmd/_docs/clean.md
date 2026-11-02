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
