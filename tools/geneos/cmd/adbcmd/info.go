package adbcmd

import (
	"archive/zip"
	"bufio"
	_ "embed"
	"fmt"
	"log/slog"
	"maps"
	"path"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/geneos/xpath"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

//go:embed _docs/info.md
var infoCmdDescription string
var infoCmdShowXPaths bool

var log *slog.Logger

func init() {
	infoCmd.Flags().BoolVarP(&infoCmdShowXPaths, "show-xpaths", "X", false, "Show a list of XPaths rather than totals for element values")

	adbCmd.AddCommand(infoCmd)
}

var infoCmd = &cobra.Command{
	Use:          "info [PATH...]",
	Short:        "Info about ADB file(s)",
	Long:         infoCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
		cmd.CmdAllowRoot:   "true",
	},
	RunE: func(command *cobra.Command, paths []string) (err error) {
		log = logger.Logger.With("command", "adb info")
		for _, file := range paths {
			log.Debug("Processing file: %s", "file", file)
			if err = infoCommand(file); err != nil {
				return
			}
		}
		return
	},
}

const (
	extension = ".Activedashboard"
	maxSize   = 1024 * 1024 * 10 // 10 MB

	fieldSeparator = '~'
)

func infoCommand(file string) (err error) {
	z, err := zip.OpenReader(file)
	if err != nil {
		return
	}
	defer z.Close()

	fmt.Printf("--- File: %q ---\n\n", file)

	// Look for the Activedashboard extension file
	for _, f := range z.File {
		log.Debug("checking zip context file: %s", "file", f.Name, "extension", path.Ext(f.Name))
		if path.Ext(f.Name) == extension {
			// Found the extension file, check its contents
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			if f.UncompressedSize64 > maxSize {
				return fmt.Errorf("file %s is too large (%d bytes)", file, f.UncompressedSize64)
			}

			scanner := bufio.NewScanner(rc)
			for scanner.Scan() {
				line := scanner.Text()
				log.Debug("Processing line: %s", "line", line)
				processLine(line)
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("error reading %s: %v", f.Name, err)
			}

			if infoCmdShowXPaths {
				fmt.Printf("File %q - XPaths: %d\n", file, len(xpaths))
				xps := slices.Sorted(maps.Keys(xpaths))
				for _, name := range xps {
					fmt.Println(name)
				}
				continue
			}

			fmt.Printf("Gateways: %d\n", len(gateways))
			gws := slices.Sorted(maps.Keys(gateways))
			for _, name := range gws {
				if name == "" {
					fmt.Println("  <empty>: ", gateways[name])
				} else {
					fmt.Printf("  %q: %d\n", name, gateways[name])
				}
			}

			ps := slices.Sorted(maps.Keys(probes))
			fmt.Printf("Probes: %d\n", len(probes))
			for _, name := range ps {
				if name == "" {
					fmt.Println("  <empty>: ", probes[name])
				} else {
					fmt.Printf("  %q: %d\n", name, probes[name])
				}
			}

			mes := slices.Sorted(maps.Keys(entities))
			fmt.Printf("Entities: %d\n", len(entities))
			for _, name := range mes {
				if name == "" {
					fmt.Println("  <empty>: ", entities[name])
				} else {
					fmt.Printf("  %q: %d\n", name, entities[name])
				}
			}

			attrs := slices.Sorted(maps.Keys(attributes))
			fmt.Printf("Attributes: %d\n", len(attributes))
			for _, name := range attrs {
				if name == "" {
					fmt.Println("  <empty>: ", len(attributes[name]))
				} else {
					fmt.Printf("  %q: %d\n", name, len(attributes[name]))
					for values := range attributes[name] {
						fmt.Printf("    =%q: %d\n", values, attributes[name][values])
					}
				}
			}

			ss := slices.Sorted(maps.Keys(samplers))
			fmt.Printf("Samplers: %d\n", len(samplers))
			for _, name := range ss {
				if name == "" {
					fmt.Println("  <empty>: ", samplers[name])
				} else {
					fmt.Printf("  %q: %d\n", name, samplers[name])
				}
			}

			ts := slices.Sorted(maps.Keys(types))
			fmt.Printf("Types: %d\n", len(types))
			for _, name := range ts {
				if name == "" {
					fmt.Println("  <empty>: ", types[name])
				} else {
					fmt.Printf("  %q: %d\n", name, types[name])
				}
			}

			dvs := slices.Sorted(maps.Keys(dataviews))
			fmt.Printf("Dataviews: %d\n", len(dataviews))
			for _, name := range dvs {
				if name == "" {
					fmt.Println("  <empty>: ", dataviews[name])
				} else {
					fmt.Printf("  %q: %d\n", name, dataviews[name])
				}
			}
		}
	}
	fmt.Println()
	return
}

var gateways = make(map[string]int)
var probes = make(map[string]int)
var entities = make(map[string]int)
var attributes = make(map[string]map[string]int)
var samplers = make(map[string]int)
var types = make(map[string]int)
var dataviews = make(map[string]int)
var xpaths = make(map[string]int)

// processLine processes a single line from an Activedashboard file and
// updates the global maps accordingly.
func processLine(line string) {
	if len(line) == 0 {
		return
	}

	fields := strings.FieldsFunc(line, func(r rune) bool {
		return r == fieldSeparator
	})
	if len(fields) == 0 {
		return
	}

	f0 := fields[0]
	if len(f0) == 0 {
		// ignore lines with empty first field
		return
	}

	if f0 == "true" || f0 == "false" {
		// an evaluation / modification line, ignore for now
		return
	}

	r, _ := utf8.DecodeRuneInString(f0)
	if unicode.IsDigit(r) {
		// ignore lines with number as the first field
		return
	}

	switch f0 {
	case mainClass:
		var mc mainClassFields
		if err := unmarshalLineToStruct(line, &mc); err != nil {
			fmt.Println("Error parsing MainClass line:", err)
			return
		}
		return
	case dataItemExprNodeClass:
		var de dataItemExprNodeFields
		if err := unmarshalLineToStruct(line, &de); err != nil {
			fmt.Println("Error parsing DataItemExprNode line:", err)
			return
		}
		dataitems := strings.Split(de.URLTarget, "dataitem(")
		if len(dataitems) == 0 {
			return
		}
		// throw away the first empty element resulting from the split
		dataitems = dataitems[1:]
		for i, d := range dataitems {
			dataitems[i] = strings.TrimSuffix(d, ")")
		}
		for _, di := range dataitems {
			xpaths[di]++
			xp, err := xpath.Parse(di)
			if err != nil {
				log.Error("Failed to parse DataitemXpath", slog.Any("error", err), slog.String("DataitemXpath", di))
			} else {
				log.Debug("Parsed DataitemXpath", slog.String("DataitemXpath", di), slog.Any("XPath", xp))
				if xp.Gateway != nil {
					gateways[xp.Gateway.Name]++
					if xp.Probe != nil {
						probes[xp.Probe.Name]++
						if xp.Entity != nil {
							entities[xp.Entity.Name]++
							for name, value := range xp.Entity.Attributes {
								if _, ok := attributes[name]; !ok {
									attributes[name] = make(map[string]int)
								}
								attributes[name][value]++
							}
							if xp.Sampler != nil {
								samplers[xp.Sampler.Name]++
								if xp.Sampler.Type != nil {
									t := *xp.Sampler.Type
									types[t]++
								}
								if xp.Dataview != nil {
									dataviews[xp.Dataview.Name]++
								}
							}
						}
					}
				}
			}
		}
	case dataItemSeverityNodeClass:
		var ds dataItemSeverityNodeFields
		if err := unmarshalLineToStruct(line, &ds); err != nil {
			fmt.Println("Error parsing DataItemSeverityNode line:", err)
			return
		}
		dataitems := strings.Split(ds.URLTarget, "dataitem(")
		if len(dataitems) == 0 {
			return
		}
		// throw away the first empty element resulting from the split
		dataitems = dataitems[1:]
		for i, d := range dataitems {
			dataitems[i] = strings.TrimSuffix(d, ")")
		}
		for _, di := range dataitems {
			xpaths[di]++
			xp, err := xpath.Parse(di)
			if err != nil {
				log.Error("Failed to parse DataitemXpath", slog.Any("error", err), slog.String("DataitemXpath", di))
			} else {
				log.Debug("Parsed DataitemXpath", slog.String("DataitemXpath", di), slog.Any("XPath", xp))
				if xp.Gateway != nil {
					gateways[xp.Gateway.Name]++
					if xp.Probe != nil {
						probes[xp.Probe.Name]++
						if xp.Entity != nil {
							entities[xp.Entity.Name]++
							for name, value := range xp.Entity.Attributes {
								if _, ok := attributes[name]; !ok {
									attributes[name] = make(map[string]int)
								}
								attributes[name][value]++
							}
							if xp.Sampler != nil {
								samplers[xp.Sampler.Name]++
								if xp.Sampler.Type != nil {
									t := *xp.Sampler.Type
									types[t]++
								}
								if xp.Dataview != nil {
									dataviews[xp.Dataview.Name]++
								}
							}
						}
					}
				}
			}
		}
	default:

	}
}
