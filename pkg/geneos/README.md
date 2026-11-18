# Geneos Configuration Model

The `geneos` package models Geneos Gateway XML configuration with Go structs. `ReadGateway` reads XML from a local path, a `file:` URL, or an HTTP(S) URL, and `WriteGateway` writes a model to a local path. These helpers do not validate a document against a Gateway XSD.

The model is intentionally partial and its structs follow the Gateway element order where that order matters, but see below for how the order of the elements of a document that is read and written is kept. Fields that have concrete models are convenient to inspect and edit. Unmodeled sections and schema choices can be kept as raw XML with `RawXMLElement`.

## Core Types

- `Gateway` is the `<gateway>` document root. Its fields represent top-level sections, including `Probes`, `ManagedEntities`, `Samplers`, `Rules`, `Environments`, `StaticVars`, and `OperatingEnvironment`.
- `Probes`, `ManagedEntities`, and `Types` describe the Gateway directory. Managed entities can refer to probes, environments, samplers, and types by name.
- `Samplers` and `Plugin` describe samplers and their plugin configuration. The root package exposes compatibility aliases for concrete plugin types, such as `DiskPlugin` and `FKMPlugin`; plugin implementations live under `pkg/geneos/plugins`.
- `Rules`, `RuleGroup`, and `Rule` model common rule fields. `RuleBlock.Statements` keeps the ordered statement elements raw because the XSD defines them as a choice of many different forms.
- `Includes` models local and remote include settings. `SamplerIncludes` models sampler references, include groups, expected rows, and the supported sampler-include plugin choices.
- `StaticVars` contains Gateway static-variable sections, including FKM tables, process descriptors, database connections, templates, and sampler schemas.
- `RawXMLElement` stores an element name, its outer attributes, and its inner XML. It is used by section-specific `Other` fields and `Gateway.Other` to preserve content that does not yet have a concrete model.

Many optional booleans and numbers are pointers. This distinguishes an omitted value from an explicit `false` or `0`.

Every struct in this package has a `Comment` field tagged as an XML comment. Leave it empty to emit no comment, or set it to include a comment inside that element.

## Build a Gateway

This example creates a probe, a managed entity that refers to it, and a rule, then writes the configuration. If the destination already exists, `Backup` preserves it before replacement:

```go
package main

import (
    "fmt"

    "github.com/itrs-group/cordial/pkg/geneos"
)

func main() {
    probePort := 7036
    gateway := geneos.Gateway{
        Compatibility: 1,
        XMLNs:         "http://www.w3.org/2001/XMLSchema-instance",
        XSI:           "http://schema.itrsgroup.com/GA7.12.1-261009/gateway.xsd",
        Probes: &geneos.Probes{
            Probes: []geneos.Probe{{
                Name:     "local",
                Hostname: "localhost",
                ProbeInfo: geneos.ProbeInfo{
                    Port: &probePort,
                },
            }},
        },
        ManagedEntities: &geneos.ManagedEntities{
            Entities: []geneos.ManagedEntity{{
                Name:  "host-1",
                Probe: &geneos.Reference{Name: "local"},
            }},
        },
        Rules: &geneos.Rules{
            Rules: []geneos.Rule{{Name: "disk-full", Priority: 1}},
        },
    }

    backup, err := geneos.WriteGateway("gateway.setup.xml", &gateway, geneos.WithGatewayBackup())
    if err != nil {
        panic(err)
    }
    if backup != "" {
        fmt.Printf("Previous setup saved to %s\n", backup)
    }
}
```

`WriteGateway` refuses to replace an existing file unless `WithGatewayBackup()` or `WithGatewayOverwrite()` is supplied. The options are mutually exclusive. Backup rotation mirrors the Gateway binary: the current file becomes `gateway.setup.xml_0001.history`, the previous `_0001` shifts to `_0002`, and so on. The default retention is 10 copies; use `WithGatewayHistoryLimit(n)` to change it. The oldest copy is pruned when the limit is reached. `WriteGateway` returns the new `_0001.history` path. `WithGatewayOverwrite()` replaces the file without creating a backup.

Read a configuration from a local path or URL with `ReadGateway`:

```go
gateway, err := geneos.ReadGateway("file:///etc/geneos/gateway.setup.xml")
if err != nil {
    return err
}
```

For HTTPS servers whose certificate cannot be verified through the normal trust store, pass `geneos.WithGatewayInsecureSkipVerify()`. This disables certificate-chain and hostname verification and makes the connection vulnerable to interception; use it only on a trusted network. `geneos.WithGatewayTimeout(duration)` sets an HTTP request deadline; a zero value leaves the request without a timeout.

```go
gateway, err := geneos.ReadGateway("https://gateway.example.test/gateway.setup.xml", geneos.WithGatewayInsecureSkipVerify())
```

Use `ReadGatewayContext` when the caller needs to cancel an HTTP read or apply its own deadline. `ReadGateway` is the convenience wrapper that uses `context.Background()`.

```go
import (
    "context"
    "time"

    "github.com/itrs-group/cordial/pkg/geneos"
)

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
gateway, err := geneos.ReadGatewayContext(ctx, "https://gateway.example.test/gateway.setup.xml")
```

## Configure a Sampler

Sampler plugin structs are assigned through the matching field on `Plugin`. Helpers such as `NewValue` and `NewSingleLineString` create Geneos literal or variable values:

```go
gateway.Samplers = &geneos.Samplers{
    Samplers: []geneos.Sampler{{
        Name:     "Disk",
        Interval: geneos.NewValue(30),
        Plugin: &geneos.Plugin{
            Disk: &geneos.DiskPlugin{
                Partitions: []geneos.Partition{{
                    Path:  geneos.NewSingleLineString("/data"),
                    Alias: geneos.NewSingleLineString("Data"),
                }},
            },
        },
    }},
}
```

`NewValue("$(threshold)")` creates a variable reference; `NewValue("30")` creates literal data. These helpers are useful for fields whose schema accepts Geneos variable expressions.

## Preserve Unmodeled XML

For schema content without a concrete type, set `XMLName` to the child element name and `Content` to that element's inner XML. Attributes are optional:

```go
gateway.Other = []geneos.RawXMLElement{{
    XMLName: xml.Name{Local: "futureSection"},
    Attributes: []xml.Attr{{
        Name:  xml.Name{Local: "enabled"},
        Value: "true",
    }},
    Content: `<setting key="mode">strict</setting>`,
}}
```

Reading, modifying and writing a Gateway setup keeps what was read. Elements that have no field in the model, at the top level or in any nested section, are read into the `Other` field of the section they are in, and attributes that have no field are read into the `OtherAttributes` field of most types. The attributes of the `gateway` element are kept in `Compatibility`, `XMLNs`, `XSI` and `Attributes`.

The model cannot hold everything, and `encoding/xml` writes the modelled fields first and the `Other` elements after them, so `ReadGateway` also remembers what is needed to write the document as it was. Not the document, but the order of the child elements of every element and, for the elements that the model would write differently from how it was read, a copy of that element. `WriteGateway` and `Gateway.MarshalOrdered` use this to rearrange and repair the marshalled XML. This is what is, and is not, kept:

* **Order of siblings.** The interleaving of siblings with different names is restored to that of the document that was read, at every level, for modelled and unmodelled elements alike. Elements are matched by their path, using the local name and the number of the earlier siblings with the same name, so the third `sampler` is matched with the third `sampler` of the original. Siblings with the same name stay in the order of the model, so a program that re-sorts or removes samplers, rules or managed entities has its order respected. An element that was added, such as a new sampler or a modelled element that was not in the document, is written directly after the sibling that comes before it in the model, or first if there is none. An element that was removed is just not written.
* **Elements the model changes.** Some older or unusual forms cannot be written back by the model as they were read, for example the `headlines`, `columns` and `rows` of dataview additions without the `var-` prefix, `<sampleInterval>80</sampleInterval>` without a `data` element, the numbers in `1000000.0`, an empty `<gatewayName/>`, and any element inside a path such as `database>mysql` that the model has no field for. If such an element has not been changed by the program, wherever it has moved to, it is written as it was read, with its original white space and comments. If it has been changed then the model writes it, in the current form, and the parts of the original that the model has no field for are lost (but see below for the empty elements). Elements that are alike, for instance `<double>0</double>` and `<double>0.0</double>`, are told apart by their position, so after a re-sort the original form of those is not always found.
* **Empty elements.** An empty element that the model leaves out, such as `<headlines/>`, `<groups/>` or `<idAttributes/>`, is put back after the elements before it if the element that it was in has been changed or moved, as long as it is clear that it is the same element: the number of siblings with the name and their attributes must be as read. Renaming an element, or changing the attributes that identify it, means that its empty elements are not put back, and the empty elements in a list of elements without attributes that has been re-sorted may be put in the wrong one.
* **Other things.** The XML declaration, and the comments and white space before and after the root element, are written back, as UTF-8 whatever the original encoding. Comments inside elements are kept in their place between the same children if the model has not written them as they were; the model keeps comments as the `Comment` field of an element, which joins them all, and that is written back if it is changed. The comments of an element that has moved are those of the model. A `gateway` without a `compatibility` attribute is written without one if `Compatibility` is zero. An `xsi:` prefix that was not declared is declared.
* **Not a document that was read.** A Gateway that was not read, such as one built in a program, is written in the order of the model. Give it the order of another with `Gateway.PreserveOrderOf` or the `WithGatewayOrderFrom` option of `WriteGateway`. `xml.Marshal` of a `Gateway` does not restore anything, use `MarshalOrdered`.
* **Failure.** If any of this fails, which it should not for a document that `encoding/xml` has written, the document is written as the model has it and `WithGatewayOrderWarning` is called with a `*GatewayOrderError`; `MarshalOrdered` returns the document with that error. The check that nothing has been lost or repeated is made before the document is returned.

### Limitations

* Only `encoding/xml` models are used, so anything below a path such as `a>b` (`database>mysql`, `listenPorts>insecure>listenPort`, `queries>query`) that is not in the model is dropped when that element is changed, as `Other` fields can only be at the level of the struct. The older `<listenPorts><insecure>7039</insecure></listenPorts>` form is read as no port.
* The older dataview additions `headlines`, `columns` and `rows`, and the text-only form of values, are written with the `var-` names and `data` elements if the element is changed. Numbers are written with Go formatting, such as `1e+06` for `1000000.0`.
* If elements have been re-sorted or removed, the comments inside them are those that the model writes, joined into one comment at the start of the element, rather than those that were read.
* A document is read with `encoding/xml`, so it must be well formed. Test templates with placeholders in the attributes, such as `[[$port]]`, and an empty `compatibility` attribute, are not read.
* Differences in white space, in the way empty elements are written (`<a/>` or `<a></a>`), in character references, in the order of attributes and in the choice of namespace prefixes between a document that was read and one that is written are not preserved unless the element is one that is written as it was read.
* The sibling order check relies on the local names of elements, so elements with the same name in different namespaces cannot be told apart.

If `Gateway.ProcessDescriptors` is set then it replaces the process descriptors in `Gateway.StaticVars` when the Gateway is written, and `StaticVars` is not changed.
