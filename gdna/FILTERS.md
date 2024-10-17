# GDNA Filters

GDNA supports filters to help organise your view of your Geneos Estate.

The follow filters are supported, and applied in this order:

* Includes
* Excludes
* Groupings
* Allocations

Include and exclude filters work with these categories of data:

* Gateways
* Servers
* Plugins
* Host IDs
* Licence Sources

Groupings work with:

* Gateways
* Servers
* Plugins

Allocations work with:

* Gateway groups

## Include Filters

Include filters are applied as the initial check for what names should be included in reports. By default all include filters have the value `*`, which is a global positive match and allows all data through.

Include filters are applied before all other types of filters.

## Exclude Filters

Exclude filters are applied next and remove matching values from reports.

## Groupings

Groupings allow you to create groups of different categories of data. These can be used to see details of your monitored estate by, for example, regions or lines-of-business and more.

## Allocations

Allocations record and display licence token allocation data against usage. Currently allocations are only used to track `server` licence tokens for Gateway groups. Allocations could also be used to track usage for specific plugins, but there is no implementation in any reports at this time.

## Managing Filters

### Filter Formats

All entries for filters are case-sensitive.

Each filter entry can be an item name or a shell-style pattern as described in the Go [`path.Match`](https://pkg.go.dev/path#Match) documentation. The most common patterns are likely to be `*` used as a prefix or suffix, e.g. `*PROD*`.

### Filter Persistence

Filters are stored in a file called `gdna-filters.json` that is located, by default, in the user's configuration directory alongside the `gdna.yaml` file. This file is loaded each time reports are run to ensure those reports always use the latest set of filters.

While this file can be updated by hand the `gdna` program provides a number of commands to automate this. In turn the `gdna.include.xml` included with the release provides a number of Active Console right-click context Commands to perform the most common filter management actions.

### Filters Dataview

The `Filters` Dataview is created as a report and so, in addition to being shown in the Active Console, can be included included in EMail XLSX reports etc.

### Context Commands

The `gdna.include.xml` file includes a variety of context Commands the manage filters. The precise disposition and availability of these commands are not yet finalised.
