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

package cmd

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
)

// startGateway serves the endpoints registered by the IMS packages,
// using the shared server in [ims.ServeWith], holding the PID file if
// there is one and reloading the configuration when told to. It does not
// return unless the server has failed, in which case the process exits.
func startGateway(cf *config.Config, pidfile string) {
	// the configuration file is recorded in the PID file, as an absolute
	// path as it is read from other directories by the control commands
	configFile := ConfigFile()
	if configFile != "" {
		if abs, err := filepath.Abs(configFile); err == nil {
			configFile = abs
		}
	}

	err := ims.ServeWith(cf, ims.Endpoints, ims.ServeOptions{
		PIDFile:    pidfile,
		ConfigFile: configFile,
		Load:       func() (*config.Config, error) { return LoadConfigFileFrom(configFile) },
	})
	if err != nil {
		log.Error("server failed", slog.Any("error", err))
		os.Exit(1)
	}
}
