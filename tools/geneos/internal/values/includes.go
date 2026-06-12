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
	return ""
}

func (i *Includes) Set(value string) error {
	if *i == nil {
		*i = Includes{}
	}
	a, b, found := strings.Cut(value, ":")

	priority := "100"
	path := a
	if found {
		priority = a
		path = b
	} else {
		// XXX check two values and first is a number
		log.Debug("second value missing after ':', using default", slog.String("priority", priority))
	}
	(*i)[priority] = path
	return nil
}

func (i *Includes) Type() string {
	return "PRIORITY:{URL|PATH}"
}
