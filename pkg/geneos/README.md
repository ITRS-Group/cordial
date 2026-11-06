# Geneos Configuration Model

The `geneos` package models Geneos Gateway XML configuration with Go structs. `ReadGateway` reads XML from a local path, a `file:` URL, or an HTTP(S) URL, and `WriteGateway` writes a model to a local path. These helpers do not validate a document against a Gateway XSD.

The model is intentionally partial and follows the Gateway element order where that order matters. Fields that have concrete models are convenient to inspect and edit. Unmodeled sections and schema choices can be kept as raw XML with `RawXMLElement`.

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

This preserves XML content semantically, not its original whitespace or formatting. Unknown top-level sections are stored in `Gateway.Other`; nested sections generally expose their own `Other` or raw-content fields.
