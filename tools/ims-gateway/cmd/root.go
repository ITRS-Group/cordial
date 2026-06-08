/*
Copyright © 2025 ITRS Group

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
	"log/slog"
	"os"

	zlog "github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
)

var configFile, Execname, logFile string
var Debug bool

func init() {
	Cmd.PersistentFlags().StringVarP(&configFile, "conf", "c", "", "override config file")

	Cmd.PersistentFlags().BoolVarP(&Debug, "debug", "d", false, "enable extra debug output")
	Cmd.PersistentFlags().MarkHidden("debug")

	// how to remove the help flag help text from the help output! Sigh...
	Cmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	Cmd.PersistentFlags().MarkHidden("help")

	Cmd.Flags().SortFlags = false

	cobra.OnInitialize(func() {
		var l slog.Level
		if Debug {
			l = slog.LevelDebug
		}
		cordial.LogInit(cordial.ExecutableName(), cordial.ToZeroLogLevel(l))
		zlog.Debug().Msgf("cordial 'ims-gateway' running as executable '%s', version %s", cordial.ExecutableName(), cordial.VERSION)
	})
}

//go:embed _docs/ims-gateway.md
var imsGatewayDescription string

var Cmd = &cobra.Command{
	Use:   "ims-gateway",
	Short: "ITRS IMS Gateway",
	Long:  imsGatewayDescription,
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
	cordial.RenderHelpAsMD(Cmd)
	err := Cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// LoadConfigFile reads in config file and ENV variables if set.
func LoadConfigFile() (cf *config.Config) {
	var err error

	cf, err = config.Read(cordial.ExecutableName(),
		config.AppName("geneos"),
		config.UseGlobal(),
		config.Format("yaml"),
		config.FilePath(configFile),
		config.MustExist(),
	)
	if err != nil {
		zlog.Fatal().Msgf("failed to load a configuration file %q from any expected location", cordial.ExecutableName()+".yaml")
	}
	zlog.Debug().Msgf("loaded config file %s",
		config.Path(cordial.ExecutableName(),
			config.AppName("geneos"),
			config.UseGlobal(),
			config.Format("yaml"),
			config.FilePath(configFile)),
	)

	return
}
