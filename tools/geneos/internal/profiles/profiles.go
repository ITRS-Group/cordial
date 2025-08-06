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

// Package profiles provides functionality to manage and manipulate profiles.

package profiles

import (
	_ "embed"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
)

//go:embed "profiles.defaults.yaml"
var profilesDefault []byte

// LoadProfiles initializes the profiles by reading from the specified configuration file.
// If the file does not exist, it creates a new one with default values.
func Load() (pf *config.Config, err error) {
	pf, err = config.Load("profiles",
		config.SetAppName(cordial.ExecutableName()),
		config.SetFileExtension("yaml"),
		config.WithDefaults(profilesDefault, "yaml"),
	)
	if err != nil {
		return
	}

	if config.Path("profiles",
		config.SetAppName(cordial.ExecutableName()),
		config.SetFileExtension("yaml"),
		config.WithDefaults(profilesDefault, "yaml"),
		config.MustExist(), // Ensure the file exists, checking if creation is required
	) == "internal defaults" {
		if err = pf.Save("profiles",
			config.SetAppName(cordial.ExecutableName()),
			config.SetFileExtension("yaml"),
		); err != nil {
			return
		}
	}
	return
}
