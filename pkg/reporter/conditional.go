/*
Copyright © 2024 ITRS Group

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

package reporter

type ConditionalFormat struct {
	Test ConditionalFormatTest  `mapstructure:"test,omitempty"`
	Set  []ConditionalFormatSet `mapstructure:"set,omitempty"`
	// Else ConditionalFormatSet   `mapstructure:"else,omitempty"`
}

type ConditionalFormatTest struct {
	Columns   []string `mapstructure:"columns,omitempty"`
	Logical   string   `mapstructure:"logical,omitempty"` // "and", "all" or "or", "any"
	Condition string   `mapstructure:"condition,omitempty"`
	Type      string   `mapstructure:"type,omitempty"`
	Value     string   `mapstructure:"value,omitempty"`
}

type ConditionalFormatSet struct {
	Rows    string   `mapstructure:"rows,omitempty"`
	NotRows string   `mapstructure:"not-rows,omitempty"`
	Columns []string `mapstructure:"columns,omitempty"`
	Format  string   `mapstructure:"format,omitempty"`
}
