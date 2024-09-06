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

package cmd

import (
	"sync"

	"github.com/itrs-group/cordial/pkg/config"
)

// HostMappings is a lookup table for substitution and other features.
type HostMappings map[string]string

// Inventories is a map of source to inventory, for "if modified" checks
var Inventories sync.Map

// LoadHosts reads the inventories and extracts the hosts and their
// types, returning them as a map of HostMappings
func LoadHosts(cf *config.Config) (hosts map[string]HostMappings, err error) {
	switch cf.GetString("inventory.type") {
	case "yaml":
		hosts, err = LoadHostsYAML(cf)
	}
	return
}
