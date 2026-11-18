/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package values

import (
	"log/slog"
	"maps"
	"slices"
	"strings"
)

// Includes is a map of include file priority to path
// include file - priority:url|path
type Includes map[string]string

// IncludeValuesOptionsText is the default help text for command to use
// for options setting include files
const IncludeValuesOptionsText = "An include file in the format `PRIORITY:[PATH|URL]`\n(Repeat as required, gateway only)"

// String is the string method for the IncludeValues type
func (i *Includes) String() string {
	if i == nil || *i == nil {
		return ""
	}
	var parts []string
	for _, k := range slices.Sorted(maps.Keys(*i)) {
		parts = append(parts, k+":"+(*i)[k])
	}
	return strings.Join(parts, ",")
}

// defaultIncludePriority is used when an include has no PRIORITY: prefix.
const defaultIncludePriority = "100"

// Set parses an include in the format [PRIORITY:]PATH|URL. The prefix
// is only taken as a priority if it is a number, so that bare URLs
// (`https://...`) and Windows paths (`C:\...`) use the default priority
// and are kept whole.
func (i *Includes) Set(value string) error {
	if *i == nil {
		*i = Includes{}
	}
	priority, path := defaultIncludePriority, value
	if a, b, found := strings.Cut(value, ":"); found && isPriority(a) {
		priority, path = a, b
	} else {
		log.Debug("no numeric priority found, using default", slog.String("priority", priority), slog.String("include", value))
	}
	(*i)[priority] = path
	return nil
}

// isPriority returns true if s is a non-empty string of decimal digits.
func isPriority(s string) bool {
	return s != "" && !strings.ContainsFunc(s, func(r rune) bool { return r < '0' || r > '9' })
}

// Type TODO: Document its behavior, inputs, and effects.
func (i *Includes) Type() string {
	return "PRIORITY:{URL|PATH}"
}
