package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var reportCmdOutDir, reportCmdOutputFile, reportCmdInstallDir, reportCmdGatewayBinary string
var reportCmdMerge bool

func init() {
	rootCmd.AddCommand(reportCmd)

	reportCmd.Flags().StringVarP(&reportCmdOutDir, "out", "o", "", "The output `DIRECTORY` (not file!)")

	reportCmd.Flags().BoolVarP(&reportCmdMerge, "merge", "m", false, "Try to create a merged config file, --gateway or --install must be set")

	reportCmd.Flags().StringVarP(&reportCmdInstallDir, "install", "i", "", "Path to the gateway installation directory")
	reportCmd.Flags().StringVarP(&reportCmdGatewayBinary, "gateway", "g", "", "Path to the gateway binary (--install can be derived from this)")

	// reportCmd.Flags().StringVarP(&reportCmdOutputFile, "output", "o", "", "The path to the output file, default write to STDOUT, ")
}

// reportCmd represents the base command when called without any subcommands
var reportCmd = &cobra.Command{
	Use:          "report [flags] [SETUP...]",
	Short:        "Report on Geneos Gateway XML files",
	Long:         ``,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var input io.Reader

		if len(args) == 0 {
			if reportCmdMerge {
				return errors.New("merge required the path to the original setup files, none given")
			}
			return generateReports(os.Stdin)
		}

		for _, setup := range args {
			if reportCmdMerge {
				if reportCmdInstallDir == "" && reportCmdGatewayBinary == "" {
					return errors.New("merge requires one of --install or --gateway to be set correctly")
				}

				setup, _ = filepath.Abs(setup)

				if reportCmdGatewayBinary != "" && reportCmdInstallDir == "" {
					reportCmdInstallDir = filepath.Dir(reportCmdGatewayBinary)
				}
				merged, err := mergeConfig(reportCmdInstallDir, reportCmdGatewayBinary, setup)
				if err != nil {
					return err
				}
				input = bytes.NewReader(merged)
			} else {
				in, err := os.Open(setup)
				if err != nil {
					return err
				}
				defer in.Close()
				input = in
			}

			if err = generateReports(input); err != nil {
				return
			}
		}
		return
	},
}

func generateReports(input io.Reader) (err error) {
	// save XML
	savedXML := new(bytes.Buffer)
	input = io.TeeReader(input, savedXML)

	gateway, entities, probes, err := processInputFile(input)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	dir := reportCmdOutDir
	if dir == "" {
		dir = cf.GetString("output.directory")
	}
	_ = os.MkdirAll(dir, 0775)

	for format, filename := range cf.GetStringMap("output.formats") {
		if filename == "" {
			continue
		}
		switch format {
		case "json":
			if err = outputJSON(cf, gateway, entities); err != nil {
				break
			}
		case "csv":
			if err = outputCSV(cf, gateway, entities, probes); err != nil {
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
			// unknown
		}
	}
	return
}
