// Package holidays provides optional Geneos holiday calendar includes.
package holidays

import "embed"

// Files contains regional holiday calendars, separate from the standard includes.
//
//go:embed files/holidays.*.xml
var Files embed.FS

// Documentation describes coverage, source confidence and calendar installation.
//
//go:embed README.md
var Documentation string
