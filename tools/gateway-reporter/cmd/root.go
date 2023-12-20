/*
Copyright © 2023 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package cmd

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

func init() {
	cobra.OnInitialize(initConfig)

	startTime = time.Now()
	startTimestamp = startTime.Format("20060102150405")

	cordial.LogInit(execname)

	RootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "config file (default is $HOME/.config/geneos/"+execname+".yaml)")

	// how to remove the help flag help text from the help output! Sigh...
	RootCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	RootCmd.PersistentFlags().MarkHidden("help")
}

func initConfig() {
	var err error
	if quiet {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	} else if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	opts := []config.FileOptions{
		config.SetAppName("geneos"),
		config.SetConfigFile(cfgFile),
		config.MergeSettings(),
		config.SetFileExtension("yaml"),
		config.WithDefaults(defaults, "yaml"),
		config.WithEnvs("GENEOS", "_"),
	}

	cf, err = config.Load(execname, opts...)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	cf.AutomaticEnv()
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
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
		initLogging(execname, "/tmp/reporter.log")
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("logging")

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

		dir := cf.GetString("output.directory")
		_ = os.MkdirAll(dir, 0775)

		for format, filename := range cf.GetStringMap("output.formats") {
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
				if err = outputCSVDir(cf, gateway, entities, probes); err != nil {
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
`, cf.GetString("output.directory"))
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the listCmd.
func Execute() {
	cordial.RenderHelpAsMD(RootCmd)

	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func initLogging(execname string, logfile string) {
	var nocolor bool
	var out io.WriteCloser
	out = os.Stderr
	if logfile != "" {
		l := &lumberjack.Logger{
			Filename:   logfile,
			MaxBackups: cf.GetInt("server.logs.backups"),
			MaxSize:    cf.GetInt("server.logs.size"),
			MaxAge:     cf.GetInt("server.logs.age"),
			Compress:   cf.GetBool("server.logs.compress"),
		}
		if cf.GetBool("server.logs.rotate-at-start") {
			l.Rotate()
		}
		out = l
		nocolor = true
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        out,
		TimeFormat: time.RFC3339,
		NoColor:    nocolor,
		FormatLevel: func(i interface{}) string {
			return strings.ToUpper(fmt.Sprintf("%s:", i))
		},
		FormatMessage: func(i interface{}) string {
			return fmt.Sprintf("%s: %s", execname, i)
		},
	})
}
