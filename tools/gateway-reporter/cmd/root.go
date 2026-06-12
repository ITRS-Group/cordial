/*
Copyright © 2023 ITRS Group

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
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
)

var cf *config.Config

var cfgFile string
var execname = cordial.ExecutableName(cordial.VERSION)
var debug, quiet bool
var startTime time.Time
var startTimestamp string

//go:embed defaults.yaml
var defaults []byte

//go:embed _docs/root.md
var rootCmdDescription string

var log = cordial.Logger

func init() {
	cobra.OnInitialize(initConfig)

	startTime = time.Now()
	startTimestamp = startTime.Format("20060102150405")

	log = cordial.LogInit(execname)

	Cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	Cmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "config file (default is $HOME/.config/geneos/"+execname+".yaml)")

	// how to remove the help flag help text from the help output! Sigh...
	Cmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	Cmd.PersistentFlags().MarkHidden("help")
}

func initConfig() {
	var err error
	opts := []config.FileOption{
		config.AppName("geneos"),
		config.FilePath(cfgFile),
		config.MergeSources(),
		config.Format("yaml"),
		config.WithDefaults(defaults, "yaml"),
		config.WithEnvs("GENEOS", "_"),
	}

	cf, err = config.Read(execname, opts...)
	if err != nil {
		log.Error("failed to read config", slog.Any("error", err))
		os.Exit(1)
	}
	cf.AutomaticEnv()
}

// Cmd represents the base command when called without any subcommands
var Cmd = &cobra.Command{
	Use:          "gateway-reporter",
	Short:        "Report on Geneos Gateway XML files",
	Long:         rootCmdDescription,
	SilenceUsage: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Version:               cordial.VERSION,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		// no logging - XML output only
		log = cordial.LogInit(execname,
			cordial.LumberjackOptions(&lumberjack.Logger{
				Filename:   "/tmp/reporter.log",
				MaxBackups: config.Get[int](cf, cf.Join("server", "logs", "backups")),
				MaxSize:    config.Get[int](cf, cf.Join("server", "logs", "size")),
				MaxAge:     config.Get[int](cf, cf.Join("server", "logs", "age")),
				Compress:   config.Get[bool](cf, cf.Join("server", "logs", "compress")),
			}),
		)
		log.Debug("logging")

		// check we are in a validate hook
		setupFile := os.Getenv("_SETUP")
		validateType := os.Getenv("_VALIDATE_TYPE")

		if setupFile == "" || (validateType != "Validate" && validateType != "Command") {
			return nil
		}

		in, err := os.Open(setupFile)
		if err != nil {
			fmt.Print(`
<validation>
<issues>
<issue>
<severity>Error</severity>
<path>/gateway</path>
<message>Cannot open setup file ` + setupFile + `</message>
</issue>
</issues>
</validation>
`)
			return err
		}
		defer in.Close()

		// save XML
		savedXML := new(bytes.Buffer)
		input := io.TeeReader(in, savedXML)

		// gateway, entities, err := processFile(config.GetConfig(), in)
		gateway, entities, probes, err := processInputFile(input)
		if err != nil {
			fmt.Printf(`
<validation>
<issues>
<issue>
<severity>Error</severity>
<path>/gateway</path>
<message>Error reading setup: %s</message>
</issue>
</issues>
</validation>
`, err)
			return err
		}

		dir := config.Get[string](cf, "output.directory")
		_ = os.MkdirAll(dir, 0775)

		for format, filename := range config.Get[map[string]any](cf, "output.formats") {
			if filename == "" {
				continue
			}
			switch format {
			case "json":
				if err = outputJSON(cf, gateway, entities, probes); err != nil {
					break
				}
			case "csv":
				if err = outputCSVZip(cf, gateway, entities, probes); err != nil {
					break
				}
			case "csvdir":
				if _, _, err = outputCSVDir(cf, gateway, entities, probes); err != nil {
					break
				}
			case "xlsx":
				if err = outputXLSX(cf, gateway, entities, probes); err != nil {
					break
				}
			case "xml":
				if err = outputXML(cf, gateway, savedXML); err != nil {
					break
				}
			default:
				//
			}
		}

		if err != nil {
			fmt.Printf(`
<validation>
<issues>
<issue>
<severity>Error</severity>
<path>/gateway</path>
<message>Error encoding report results: %s</message>
</issue>
</issues>
</validation>
`, err)
			return err
		}

		fmt.Printf(`
<validation>
<issues>
<issue>
<severity>None</severity>
<path>/gateway</path>
<message>Report file(s) written to %q</message>
</issue>
</issues>
</validation>
`, config.Get[string](cf, "output.directory"))
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the listCmd.
func Execute() {
	cordial.RenderHelpAsMD(Cmd)

	err := Cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
