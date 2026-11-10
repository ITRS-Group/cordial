# ADB embedded target and link notes

## What the command currently handles

`info.go` parses each top-level `dataitem(...)` expression in modifier targets, including mixed target fields. Its `--show-xpaths`, `--show-urls`, and `--show-all` modes list sorted, unique values; URL destinations are flattened without link-type or source labels.

`clean.go` applies XPath actions to `dataitem(...)` expressions and regexp replacements to URLs inside `wwwpage(...)` expressions. Use repeatable `--filter-url REGEXP` flags to select web URLs; any matching regexp admits a URL. These actions work for both `GeneosURL` records and mixed modifier targets, while unrelated expressions are preserved. Each `GeneosURL` record is handled as one target, not a concatenated list.

## Target and link forms

| Record class | Form | Meaning / example |
| --- | --- | --- |
| `GeneosURL` | `dataitem(...)` | Link to a Geneos XPath target. |
| `GeneosURL` | `wwwpage(...)` | Web link whose argument is a URL. |
| `GeneosURL` | `geneos://...` | Active Console navigation link, commonly `component(ac)/dockable(...)/tab(...)`. |
| `DataItemExprNode` | `dataitem(...)` | Modifier expression referencing a Geneos XPath. |
| `DataItemExprNode` | `droppedItem()` | Modifier expression referring to the item dropped onto the dashboard. |
| `DataItemExprNode` | `droppeditem()/replace(...)` | Dropped-item expression with a path replacement operation. |
| `DataItemSeverityNode` | `dataitem(...)` | Severity modifier referencing a Geneos XPath. Some use `param(...)` predicates. |
| `DataItemSeverityNode` | `droppedItem()` | Severity modifier referring to the dropped item. |
| `DataItemSeverityNode` | `geneos://...` | Navigation target also present in severity records. |
| `DataItemSeverityNode` | dropped-item path operations | `droppeditem()/removeafter(...)/append(...)` form. |

## Example record excerpts

Examples are anonymized. Fields are separated by `~`; `...` omits fields unrelated to the target.

### Geneos dataitem link

From `example-1.adb`:

```text
GeneosURL~3.0~dataitem(/geneos/gateway/directory/probe/managedEntity/sampler[(@name="Entity 1")]/dataview[(@name="View 1")])~Example View~
```

### Web page link

From `example-2.adb`:

```text
GeneosURL~3.0~wwwpage(https://example.invalid/dashboard)~https://example.invalid/dashboard~
```

The `wwwpage(...)` wrapper is distinct from the URL stored as the link's display/name field.

### Active Console navigation link

From `example-3.adb`:

```text
GeneosURL~3.0~geneos://component(ac)/dockable(EXAMPLE)/tab(EXAMPLE | OVERVIEW)~geneos:// ac / EXAMPLE / EXAMPLE | OVERVIEW~
```

### Modifier dataitem target

From `example-4.adb`:

```text
DataItemExprNode~7.0~$0$~Value~dataitem(/geneos/gateway/directory/probe/managedEntity/sampler[(@name="Entity 2")]/dataview[(@name="View 2")]/rows/row[(@name="1")]/cell[(@column="field-1")])~...
```

### Dropped-item expression

From `example-5.adb`:

```text
DataItemExprNode~7.0~$0$~Snoozed~droppedItem()~droppedItem~...
```

### Dropped-item path operations

From `example-6.adb`:

```text
DataItemExprNode~7.0~$0$~Value~droppeditem()/replace(/cell[(@column="field-2")])~...
```

From `example-7.adb`:

```text
DataItemSeverityNode~7.0~$0$~Severity~droppeditem()/removeafter(/dataview)/append(/headlines/cell[wild(@name,"example_*")])~...
```

### Severity dataitem with parameters

From `example-8.adb`:

```text
DataItemSeverityNode~7.0~$0$~Severity~dataitem(/geneos/gateway[(@name="Gateway 1")]/directory/probe[(param("Group")="Probe Group 1")]/managedEntity[(param("Group")="Entity Group 1")])~...
```

## Practical distinction

Treat `dataitem(...)` as a Geneos XPath reference, `wwwpage(...)` as a web destination, and `geneos://...` as Active Console navigation. `droppedItem()` and its `/replace`, `/removeafter`, and `/append` variants are modifier expressions, not ordinary links. `info` parses dataitem references and reports GeneosURL target types; `clean` updates dataitem paths and web destinations while preserving other modifier expressions.
