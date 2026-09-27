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

// attribute - name=value
type NameValues []string

const AttributesOptionsText = "Attribute in the format `NAME=VALUE\n(Repeat as required, san only)"
const EnvsOptionsText = "Environment variable for instance start-up in the format NAME=VALUE\n(Repeat as required)"
const HeadersOptionsText = "HTTP header in the format NAME=VALUE\n(Repeat as required)"

func (i *NameValues) String() string {
	return ""
}

func (i *NameValues) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *NameValues) Type() string {
	return "NAME=VALUE"
}
