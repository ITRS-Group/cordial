Clean ADB files, currently applicable only to dataitem XPaths in limited, but common, scenarios.

Given one or more Active Dashboard files (normally with an `.adb` extension) this command will apply the specified modifications and create an updated version of each file, by default with the following elements having their name predicates removed: "gateway", "probe" and "entity". With command line flags you can exercise more granular control over which elements to keep, remove, or modify. The original file is saved with a `.bak` extension and a timestamp, the new file replaces the original.

If you use the `--keep-all`/`-k` flag, all elements will be kept regardless of other cleaning options, overriding the default behavior. This can be combined with `--filter` to selectively apply the modifications to specific XPaths and modifiers.

The `--filter` flag allows you to specify a XPath to selectively apply the cleaning modifications to matching dataitems within the Active Dashboard files. The strings in the filter are applied as "globs", i.e. like shell wildcards. Only the elements matching the filter will be affected by the cleaning options. The filter is only applied to the level of the XPath specified and does not affect lower level elements. For example `--filter '//managedEntity[@name="EntityName"]'` will only apply the selected modifications to dataitems that match an entity named "EntityName", while the filter `--filter '//gateway[(@name="PROD*")]'` will apply the modifications to dataitems that match a gateway with a name starting with "PROD".

To control which elements have their name predicates removed, use the available command line flags for each element type to specify the elements to keep, remove, or modify. Each option in in the form `--element MODIFIER`, e.g. `--gateway set:GATEWAY_NAME`. Each flag can be repeated as many times as required.

For "gateway", "probe", "entity", "sampler", "type" and "dataview" elements, the following modifications are available:

* `remove-all` - remove the name predicate from the element. This is the default for "gateway", "probe", and "entity" elements.
* `keep` - keep the name predicate for the element. This is the default for "sampler", "type", and "dataview" elements.
* `remove:PATTERN` - remove the matching name predicate from the element. The pattern must be a valid Go regular expression, and because of this you may need to anchor it with `^` and `$` if you want to match the entire name.
* `set:NAME` - set the name predicates for the element.
* `replace:/pattern/replacement/` - replace the matching pattern, as many times as it appears, with the replacement string. The pattern must be a valid Go regular expression. The replacement can only be a fixed string at this time, and placeholders are not supported. The '/' delimiter can be any character you choose but must be provided at the start and end of the expression as well as a separator between `pattern` and `replacement`. The delimiter can be escaped in either the pattern or the replacement using a backslash.

For "attributes" elements, the following modifications are available:

* `remove-all` - remove all attributes from the managed entity element.
* `remove:NAME` - remove specific attributes from the managed entity element.
* `add:NAME=VALUE` - add specific attributes for the managed entity element.
* `replace:NAME=VALUE` - replace the value of the specified attribute with the given value. Unlike `add:NAME=VALUE`, this will only update the value if the attribute already exists.
* `replace:NAME=/PATTERN/REPLACEMENT/` - replace the matching pattern in the attribute value with the replacement string. The pattern must be a valid Go regular expression. The replacement can only be a fixed string at this time, and placeholders are not supported. The delimiter must be a '/' and must appear at the start and end of the expression as well as a separator between `PATTERN` and `REPLACEMENT`. A '/' can be escaped in either the pattern or the replacement using a backslash. Only existing attributes with the specified NAME will be updated.

The modifications are applied in the order given on the command line and there is no priority or precedence among them.
