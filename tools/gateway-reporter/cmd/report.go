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
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/geneos"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var reportCmdOutDir, reportCmdPrefix, reportCmdInstallation string
var reportCmdMerge bool

func init() {
	RootCmd.AddCommand(reportCmd)

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
		var input io.Reader

		if len(args) == 0 {
			args = []string{"-"}
		}

		if reportCmdMerge && reportCmdInstallation == "" {
			return errors.New("merge requires --install DIR|FILE to be set correctly")
		}

		i := 1
		prefix := reportCmdPrefix

		for _, setup := range args {
			if reportCmdPrefix != "" && len(args) > 1 {
				prefix = fmt.Sprintf("%s_%d", reportCmdPrefix, i)
				i++
			}
			switch {
			case setup == "-":
				if reportCmdMerge {
					return errors.New("merge required the path to the setup files, none given")
				}
				input = os.Stdin
			case strings.HasPrefix(setup, "https://") || strings.HasPrefix(setup, "http://"):
				if reportCmdMerge {
					merged, err := mergeConfig(reportCmdInstallation, setup)
					if err != nil {
						return err
					}
					input = bytes.NewReader(merged)
					break
				}
				resp, err := http.Get(setup)
				if err != nil {
					return err
				}
				if resp.StatusCode > 299 {
					resp.Body.Close()
					return fmt.Errorf("failed to fetch %s - %s", setup, resp.Status)
				}
				input = resp.Body
			case strings.HasPrefix(setup, "~/"):
				home, _ := config.UserHomeDir()
				setup = filepath.Join(home, strings.TrimPrefix(setup, "~/"))
				fallthrough
			default:
				if reportCmdMerge {
					setup, _ = filepath.Abs(setup)
					merged, err := mergeConfig(reportCmdInstallation, setup)
					if err != nil {
						return err
					}
					input = bytes.NewReader(merged)
					break
				}
				in, err := os.Open(setup)
				if err != nil {
					return err
				}
				defer in.Close()
				input = in
			}

			gateway, err := generateReports(input, prefix)
			if err != nil {
				return err
			}
			fmt.Printf("Report(s) for gateway %q written to directory %s\n", gateway, cf.GetString("output.directory"))
		}
		return
	},
}

func generateReports(input io.Reader, prefix string) (gateway string, err error) {
	// save XML
	savedXML := new(bytes.Buffer)
	input = io.TeeReader(input, savedXML)

	gateway, entities, probes, err := processInputFile(input)
	if err != nil {
		log.Error().Err(err).Msg("")
	}
	if gateway == "" {
		if prefix == "" {
			err = errors.New("no gateway name found in configuration. perhaps you wanted to use --merge?")
			return
		}
		gateway = prefix
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
			if err = outputJSON(cf, gateway, entities, probes); err != nil {
				break
			}
		case "csv":
			if err = outputCSVZip(cf, gateway, entities, probes); err != nil {
				break
			}
		case "csvdir":
			csvFiles, destdir, err := outputCSVDir(cf, gateway, entities, probes)
			if err != nil {
				break
			}
			if cf.GetBool("output.toolkit-include.enable") {
				log.Debug().Msg("building include")
				err = outputToolkitInclude(cf, gateway, destdir, csvFiles)
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

func outputToolkitInclude(cf *config.Config, gateway string, destdir string, csvFiles []csvFiles) (err error) {
	conftable := config.LookupTable(map[string]string{
		"gateway":  gateway,
		"datetime": startTimestamp,
	})

	includeFile := cf.GetString("output.toolkit-include.include-file", conftable)
	if !filepath.IsAbs(includeFile) {
		includeFile = filepath.Join(destdir, includeFile)
	}
	log.Debug().Msgf("include: %s", includeFile)

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
			Name:            cf.GetString("output.toolkit-include.sampler-name", samplerConftable),
			SampleOnStartup: true,
			Group:           geneos.NewSingleLineString(cf.GetString("output.toolkit-include.sampler-group", samplerConftable)),
			Plugin: &geneos.Plugin{Toolkit: &geneos.ToolkitPlugin{
				SamplerScript: geneos.NewSingleLineString(cf.GetString("output.toolkit-include.sampler-script", samplerConftable)),
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
				Name:     cf.GetString("output.toolkit-include.entity-name", conftable),
				Probe:    &geneos.Reference{Name: cf.GetString("output.toolkit-include.probe-name", conftable)},
				Samplers: samplerRefs,
			}},
		},
		Samplers: &geneos.Samplers{Samplers: samplers},
	}

	outfile, err := os.CreateTemp(filepath.Dir(includeFile), filepath.Base(includeFile))
	if err != nil {
		return
	}
	tmpfile := outfile.Name()
	defer os.Remove(tmpfile)
	defer outfile.Close()

	fmt.Fprint(outfile, xml.Header)
	b, err := xml.MarshalIndent(include, "", "    ")
	if err != nil {
		return
	}
	if _, err = outfile.Write(b); err != nil {
		return
	}
	// windows cannot rename an open file
	// this is a small race condition, but otherwise it doesn't work
	if runtime.GOOS == "windows" {
		outfile.Close()
	}
	if err = os.Rename(tmpfile, includeFile); err != nil {
		return
	}
	return
}
