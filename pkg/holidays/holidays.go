// Package holidays provides optional Geneos holiday calendar includes.
package holidays

import (
	"embed"
	"strings"
)

//go:embed VERSION
var version string

// VERSION records the source-fetch date of this holiday snapshot as vYYYY.MM.DD.
var VERSION = strings.TrimSpace(version)

// Files contains regional holiday calendars, separate from the standard includes.
//
//go:embed files/holidays.*.xml
var Files embed.FS

// Documentation describes coverage, source confidence and calendar installation.
//
//go:embed README.md
var Documentation string
