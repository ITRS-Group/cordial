Clean ADB files, currently applicable only to dataitem XPaths in limited, but common, scenarios.

Given one or more Active Dashboard files (normally with an `.adb` extension) this command will apply the specified modifications and create an updated version of each file, by default with the following elements having their name predicates removed: "gateway", "probe" and "entity". With command line flags you can exercise more granular control over which elements to keep, remove, or modify.

By default, the original file is saved (in the same location as the source file) with a timestamp added before the file extension, the processed file replaces the original. If you specify an output directory using `--output`/`-o`, the processed file will be written there instead, and no backup of the original will be made unless the directory is the same as the for the source.

If you use the `--keep-all`/`-k` flag, the default action for all elements will be `keep`, overriding the default behavior. This can be combined with `--filter` to selectively apply the modifications to specific XPaths and modifiers.

The `--filter-*` flags allows you to specify either an XPath or specific component matches to selectively apply the cleaning modifications to matching dataitems within the Active Dashboard files. The strings in the filter are applied as "globs", i.e. like shell wildcards. Only the elements matching the filter will be affected by the cleaning options.

Each filter is applied in the order shown in the command usage, namely: `--filter-xpath`, `--filter-gateway`, `--filter-probe`, `--filter-entity`, `--filter-sampler`, `--filter-dataview`, and `--filter-type`.

To control which elements have their name predicates removed, use the available command line flags for each element type to specify the elements to keep, remove, or modify. Each option in in the form `--element MODIFIER`, e.g. `--gateway set:GATEWAY_NAME`. Each flag can be repeated as many times as required.

For "gateway", "probe", "entity", "sampler", and "dataview" elements, the following modifications are available:

* `remove-all` - remove the name predicate from the element. This is the default for "gateway", "probe", and "entity" elements.
* `keep` - keep the name predicate for the element. This is the default for "sampler", "type", and "dataview" elements.
* `remove:NAME` - remove the matching name predicate from the element. NAME can be a glob (not regexp) pattern, e.g. "PROD*".
* `set:NAME` - set the name predicates for the element.
* `replace:/pattern/replacement/` - replace the matching pattern, as many times as it appears, with the replacement string. The pattern must be a valid Go regular expression. The replacement can only be a fixed string at this time, and placeholders are not supported. The '/' delimiter can be any character you choose but must be provided at the start and end of the expression as well as a separator between `pattern` and `replacement`. The delimiter can be escaped in either the pattern or the replacement using a backslash.

For "type" elements, the following additional modifications are available:

* `remove-empty` - remove the type if it has an empty name.
* `set-empty` - set the type to have an empty name. Note that this is different from no type.

For "attributes" elements, the following modifications are available:

* `remove-all` - remove all attributes from the managed entity element.
* `remove:NAME` - remove specific attributes from the managed entity element.
* `add:NAME=VALUE` - add specific attributes for the managed entity element.
* `replace:NAME=VALUE` - replace the value of the specified attribute with the given value. Unlike `add:NAME=VALUE`, this will only update the value if the attribute already exists.
* `replace:NAME=/PATTERN/REPLACEMENT/` - replace the matching pattern in the attribute value with the replacement string. The pattern must be a valid Go regular expression. The replacement can only be a fixed string at this time, and placeholders are not supported. The delimiter must be a '/' and must appear at the start and end of the expression as well as a separator between `PATTERN` and `REPLACEMENT`. A '/' can be escaped in either the pattern or the replacement using a backslash. Only existing attributes with the specified NAME will be updated.

The modifications are applied in the order given on the command line and there is no priority or precedence among them.
