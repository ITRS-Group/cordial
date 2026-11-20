// Package includes provides the standard Geneos Gateway include corpus.
package includes

import (
	"embed"
	"strings"
)

//go:embed VERSION
var version string

// VERSION is the standard include corpus version, independent of Cordial.
var VERSION = strings.TrimSpace(version)

// Files contains read-only functional includes and editable configuration templates.
//
//go:embed files/itrs.*.xml files/config.*.xml
var Files embed.FS

// Documentation describes installation and use of the standard includes.
//
//go:embed README.md
var Documentation string
