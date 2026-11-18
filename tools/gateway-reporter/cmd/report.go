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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos"
)

var reportCmdOutDir, reportCmdPrefix, reportCmdInstallation string
var reportCmdMerge bool

func init() {
	Cmd.AddCommand(reportCmd)

	reportCmd.Flags().StringVarP(&reportCmdOutDir, "out", "o", "", "Write reports to `DIRECTORY`. Default `/tmp/gateway-reporter`")

	reportCmd.Flags().BoolVarP(&reportCmdMerge, "merge", "m", false, "Create a merged config file. --install must be set")

	reportCmd.Flags().StringVarP(&reportCmdInstallation, "install", "i", "", "Path to the gateway installation `BINARY|DIR`")

	reportCmd.Flags().StringVarP(&reportCmdPrefix, "prefix", "p", "", "Report prefix for configurations without a Gateway `name`")

	reportCmd.Flags().SortFlags = false
}

//go:embed _docs/report.md
var reportCmdDescription string

// reportCmd represents the base command when called without any subcommands
var reportCmd = &cobra.Command{
	Use:          "report [flags] [SETUP...]",
	Short:        "Report on Geneos Gateway XML files",
	Long:         reportCmdDescription,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) == 0 {
			args = []string{"-"}
		}

		if reportCmdMerge && reportCmdInstallation == "" {
			return errors.New("merge requires --install DIR|FILE to be set correctly")
		}

		if reportCmdOutDir != "" {
			config.Set(cf, "output.directory", reportCmdOutDir)
		}

		i := 1
		prefix := reportCmdPrefix

		for _, setup := range args {
			if reportCmdPrefix != "" && len(args) > 1 {
				prefix = fmt.Sprintf("%s_%d", reportCmdPrefix, i)
				i++
			}

			gateway, err := reportSetup(setup, prefix)
			if err != nil {
				return err
			}
			fmt.Printf("Report(s) for gateway %q written to directory %s\n", gateway, config.Get[string](cf, "output.directory"))
		}
		return
	},
}

// httpClient is used to fetch remote setup files
var httpClient = &http.Client{Timeout: 5 * time.Minute}

// reportSetup opens the setup source (STDIN, a URL or a file), merging
// it first if required, and generates the reports for it
func reportSetup(setup, prefix string) (gateway string, err error) {
	var input io.Reader

	switch {
	case setup == "-":
		if reportCmdMerge {
			return "", errors.New("merge required the path to the setup files, none given")
		}
		input = os.Stdin
	case strings.HasPrefix(setup, "https://") || strings.HasPrefix(setup, "http://"):
		if reportCmdMerge {
			merged, err := mergeConfig(reportCmdInstallation, setup)
			if err != nil {
				return "", err
			}
			input = bytes.NewReader(merged)
			break
		}
		resp, err := httpClient.Get(setup)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode > 299 {
			return "", fmt.Errorf("failed to fetch %s - %s", setup, resp.Status)
		}
		input = resp.Body
	default:
		setup = config.ResolveHome(setup)
		if reportCmdMerge {
			if setup, err = filepath.Abs(setup); err != nil {
				return "", err
			}
			merged, err := mergeConfig(reportCmdInstallation, setup)
			if err != nil {
				return "", err
			}
			input = bytes.NewReader(merged)
			break
		}
		in, err := os.Open(setup)
		if err != nil {
			return "", err
		}
		defer in.Close()
		input = in
	}

	return generateReports(input, prefix)
}

// xmlFormatEnabled returns true if the merged XML is to be saved as one
// of the output formats
func xmlFormatEnabled(cf *config.Config) bool {
	return config.Get[string](cf, "output.formats.xml") != ""
}

func generateReports(input io.Reader, prefix string) (gateway string, err error) {
	// only keep a copy of the XML if it will be written out
	savedXML := new(bytes.Buffer)
	if xmlFormatEnabled(cf) {
		input = io.TeeReader(input, savedXML)
	}

	gateway, entities, probes, err := processInputFile(input)
	if err != nil {
		log.Error("failed to process input file", slog.Any("error", err))
		return "", err
	}
	if gateway == "" {
		if prefix == "" {
			err = errors.New("no gateway name found in configuration. perhaps you wanted to use --merge?")
			return
		}
		gateway = prefix
	}

	err = writeReports(cf, gateway, entities, probes, savedXML)
	return
}

// writeReports writes all the enabled output formats, in a stable order,
// and returns the first error encountered. The toolkit include file is
// written if the csvdir format and the include are both enabled.
func writeReports(cf *config.Config, gateway string, entities []Entity, probes map[string]geneos.Probe, savedXML *bytes.Buffer) (err error) {
	dir := config.Get[string](cf, "output.directory")
	if err = os.MkdirAll(dir, 0775); err != nil {
		return
	}

	formats := config.Get[map[string]any](cf, "output.formats")
	for _, format := range slices.Sorted(maps.Keys(formats)) {
		if filename, _ := formats[format].(string); filename == "" {
			continue
		}
		switch format {
		case "json":
			err = outputJSON(cf, gateway, entities, probes)
		case "csv":
			err = outputCSVZip(cf, gateway, entities, probes)
		case "csvdir":
			var csvFiles []csvFiles
			var destdir string
			csvFiles, destdir, err = outputCSVDir(cf, gateway, entities, probes)
			if err != nil {
				break
			}
			if config.Get[bool](cf, cf.Join("output", "toolkit-include", "enable")) {
				log.Debug("building include")
				if err = outputToolkitInclude(cf, gateway, destdir, csvFiles); err != nil {
					log.Error("failed to build toolkit include", slog.Any("error", err))
				}
			}
		case "xlsx":
			err = outputXLSX(cf, gateway, entities, probes)
		case "xml":
			err = outputXML(cf, gateway, savedXML)
		default:
			// unknown
		}
		if err != nil {
			return fmt.Errorf("%s output: %w", format, err)
		}
	}
	return
}

func outputToolkitInclude(cf *config.Config, gateway string, destdir string, csvFiles []csvFiles) (err error) {
	conftable := config.LookupTable(map[string]string{
		"gateway":  gateway,
		"datetime": startTimestamp,
	})

	includeFile := config.Get[string](cf, "output.toolkit-include.include-file", conftable)
	if !filepath.IsAbs(includeFile) {
		includeFile = filepath.Join(destdir, includeFile)
	}
	log.Debug("include file", slog.String("includeFile", includeFile))

	samplers := make([]geneos.Sampler, len(csvFiles))
	for i, c := range csvFiles {
		samplerConftable := config.LookupTable(map[string]string{
			"gateway":   gateway,
			"datetime":  startTimestamp,
			"csvfile":   c.path,
			"filename":  c.filename,
			"sheetname": c.sheetname,
		})

		samplers[i] = geneos.Sampler{
			Name:            config.Get[string](cf, "output.toolkit-include.sampler-name", samplerConftable),
			SampleOnStartup: new(true),
			Group:           geneos.NewSingleLineString(config.Get[string](cf, "output.toolkit-include.sampler-group", samplerConftable)),
			Plugin: &geneos.Plugin{Toolkit: &geneos.ToolkitPlugin{
				SamplerScript: geneos.NewSingleLineString(config.Get[string](cf, "output.toolkit-include.sampler-script", samplerConftable)),
			}},
		}
	}

	samplerRefs := []geneos.SamplerRef{}
	for _, s := range samplers {
		samplerRefs = append(samplerRefs, geneos.SamplerRef{
			Name: s.Name,
		})
	}
	include := geneos.Gateway{
		Compatibility: 1,
		XMLNs:         "http://www.w3.org/2001/XMLSchema-instance",
		XSI:           "http://schema.itrsgroup.com/GA5.14.2-220707/gateway.xsd",
		ManagedEntities: &geneos.ManagedEntities{
			Entities: []geneos.ManagedEntity{{
				Name:     config.Get[string](cf, "output.toolkit-include.entity-name", conftable),
				Probe:    &geneos.Reference{Name: config.Get[string](cf, "output.toolkit-include.probe-name", conftable)},
				Samplers: samplerRefs,
			}},
		},
		Samplers: &geneos.Samplers{Samplers: samplers},
	}

	b, err := xml.MarshalIndent(include, "", "    ")
	if err != nil {
		return
	}

	outfile, err := os.CreateTemp(filepath.Dir(includeFile), filepath.Base(includeFile))
	if err != nil {
		return
	}
	tmpfile := outfile.Name()
	// no-op after a successful rename
	defer os.Remove(tmpfile)

	_, err = outfile.WriteString(xml.Header + string(b))
	if err == nil {
		// CreateTemp uses 0600, match the other report files
		err = outfile.Chmod(0644)
	}
	if cerr := outfile.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return
	}
	return os.Rename(tmpfile, includeFile)
}
