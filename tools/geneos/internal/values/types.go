/*
Copyright © 2026 ITRS Group

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

import "strings"

// attribute - name=value
type Types []string

// TypesOptionsText TODO: Describe its purpose and intended use.
const TypesOptionsText = "A type NAME\n(Repeat as required, san only)"

// String TODO: Document its behavior, inputs, and effects.
func (i *Types) String() string {
	return strings.Join(*i, ",")
}

// Set TODO: Document its behavior, inputs, and effects.
func (i *Types) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// Type TODO: Document its behavior, inputs, and effects.
func (i *Types) Type() string {
	return "NAME"
}
