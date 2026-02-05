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
	_ "embed"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
)

var configFile string
var Debug bool

//go:embed sdp.defaults.yaml
var defaultConfig []byte

var RootCmd = &cobra.Command{
	Use:   "sdp",
	Short: "Geneos to ServiceDesk Plus integration",
	Long:  ``,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Version:           cordial.VERSION,
	DisableAutoGenTag: true,
	SilenceUsage:      true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cordial.RenderHelpAsMD(RootCmd)
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// LoadConfigFile reads in config file and ENV variables if set.
func LoadConfigFile() (cf *config.Config) {
	var err error

	cf, err = config.Load(cordial.ExecutableName(),
		config.SetAppName("geneos"),
		config.UseGlobal(),
		config.SetFileExtension("yaml"),
		config.SetConfigFile(configFile),
		config.WithDefaults(defaultConfig, "yaml"),
		config.MustExist(),
	)
	if err != nil {
		log.Fatal().Msgf("failed to load a configuration file %q from any expected location", cordial.ExecutableName()+".yaml")
	}
	log.Debug().Msgf("loaded config file %s",
		config.Path(cordial.ExecutableName(),
			config.SetAppName("geneos"),
			config.UseGlobal(),
			config.SetFileExtension("yaml"),
			config.WithDefaults(defaultConfig, "yaml"),
			config.SetConfigFile(configFile)),
	)

	return
}
