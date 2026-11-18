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
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DeRuina/timberjack"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
)

var cf *config.Config

var cfgFile string
var execname = cordial.ExecutableName(cordial.VERSION)
var debug bool

// TODO: Revisit unused declaration quiet; restore it only when needed.
// var quiet bool
var startTime time.Time
var startTimestamp string

//go:embed defaults.yaml
var defaults []byte

//go:embed _docs/root.md
var rootCmdDescription string

var log = logger.Logger

func init() {
	cobra.OnInitialize(initConfig)

	startTime = time.Now()
	startTimestamp = startTime.Format("20060102150405")

	log = logger.Init(execname)

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
		config.StopOnInternalDefaultsErrors(),
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
		// check we are in a validate hook
		setupFile := os.Getenv("_SETUP")
		validateType := os.Getenv("_VALIDATE_TYPE")

		if setupFile == "" || (validateType != "Validate" && validateType != "Command") {
			return nil
		}

		// no logging to the console - XML output only
		logs := &timberjack.Logger{
			Filename:   filepath.Join(os.TempDir(), "reporter.log"),
			MaxBackups: config.Get[int](cf, cf.Join("server", "logs", "backups")),
			MaxSize:    config.Get[int](cf, cf.Join("server", "logs", "size")),
			MaxAge:     config.Get[int](cf, cf.Join("server", "logs", "age")),
			Compress:   config.Get[bool](cf, cf.Join("server", "logs", "compress")),
		}
		defer logs.Close()
		logger.Init(execname, logger.SetWriter(logs))
		log.Debug("logging")

		in, err := os.Open(setupFile)
		if err != nil {
			printValidation("Error", "Cannot open setup file "+setupFile)
			return err
		}
		defer in.Close()

		// only keep a copy of the XML if it will be written out
		var input io.Reader = in
		savedXML := new(bytes.Buffer)
		if xmlFormatEnabled(cf) {
			input = io.TeeReader(in, savedXML)
		}

		gateway, entities, probes, err := processInputFile(input)
		if err != nil {
			printValidation("Error", "Error reading setup: "+err.Error())
			return err
		}

		if err = writeReports(cf, gateway, entities, probes, savedXML); err != nil {
			printValidation("Error", "Error encoding report results: "+err.Error())
			return err
		}

		printValidation("None", fmt.Sprintf("Report file(s) written to %q", config.Get[string](cf, "output.directory")))
		return nil
	},
}

// printValidation writes a Geneos validation hook response with a
// single issue to STDOUT
func printValidation(severity, message string) {
	var msg strings.Builder
	xml.EscapeText(&msg, []byte(message))
	fmt.Printf(`
<validation>
<issues>
<issue>
<severity>%s</severity>
<path>/gateway</path>
<message>%s</message>
</issue>
</issues>
</validation>
`, severity, msg.String())
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
