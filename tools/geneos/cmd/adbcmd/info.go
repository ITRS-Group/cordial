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
var infoCmdShowXPaths, infoCmdShowURLs, infoCmdShowAll bool

func init() {
	infoCmd.Flags().BoolVarP(&infoCmdShowXPaths, "show-xpaths", "X", false, "List XPaths only")
	infoCmd.Flags().BoolVar(&infoCmdShowURLs, "show-urls", false, "List URL targets only")
	infoCmd.Flags().BoolVar(&infoCmdShowAll, "show-all", false, "List both XPaths and URL targets")

	adbCmd.AddCommand(infoCmd)
}

var infoCmd = &cobra.Command{
	Use:          "info [PATH...]",
	Short:        "Info about ADB file(s)",
	Long:         infoCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:       "false",
		cmd.CmdRequireHome:  "false",
		cmd.CmdAllowRoot:    "true",
		cmd.CmdAuditCommand: "never",
		cmd.CmdAuditActions: "never",
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
				// log.Debug("Processing line: %s", "line", line)
				processLine(line)
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("error reading %s: %v", f.Name, err)
			}

			if infoCmdShowXPaths || infoCmdShowURLs || infoCmdShowAll {
				fmt.Print(formatTargetLists(
					infoCmdShowXPaths || infoCmdShowAll,
					infoCmdShowURLs || infoCmdShowAll,
					xpaths,
					urls,
				))
				continue
			}

			fmt.Printf("Gateways: %d\n", len(gateways))
			gws := slices.Sorted(maps.Keys(gateways))
			for _, name := range gws {
				if name == "" {
					fmt.Println("  <no name>: ", gateways[name])
				} else {
					fmt.Printf("  %q: %d\n", name, gateways[name])
				}
			}

			ps := slices.Sorted(maps.Keys(probes))
			fmt.Printf("Probes: %d\n", len(probes))
			for _, name := range ps {
				if name == "" {
					fmt.Println("  <no name>: ", probes[name])
				} else {
					fmt.Printf("  %q: %d\n", name, probes[name])
				}
			}

			mes := slices.Sorted(maps.Keys(entities))
			fmt.Printf("Entities: %d\n", len(entities))
			for _, name := range mes {
				if name == "" {
					fmt.Println("  <no name>: ", entities[name])
				} else {
					fmt.Printf("  %q: %d\n", name, entities[name])
				}
			}

			attrs := slices.Sorted(maps.Keys(attributes))
			fmt.Printf("Attributes: %d\n", len(attributes))
			for _, name := range attrs {
				if name == "" {
					fmt.Println("  <no name>: ", len(attributes[name]))
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
					fmt.Println("  <no name>: ", samplers[name])
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
			linkTypes := slices.Sorted(maps.Keys(links))
			linkCount := 0
			for _, name := range linkTypes {
				linkCount += links[name]
			}
			fmt.Printf("Links: %d\n", linkCount)

			dvs := slices.Sorted(maps.Keys(dataviews))
			fmt.Printf("Dataviews: %d\n", len(dataviews))
			for _, name := range dvs {
				if name == "" {
					fmt.Println("  <no name>: ", dataviews[name])
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
var links = make(map[string]int)
var urls = make(map[string]int)

func formatTargetLists(showXPaths, showURLs bool, xpaths, urls map[string]int) string {
	var output strings.Builder
	if showXPaths {
		fmt.Fprintf(&output, "XPaths: %d\n", len(xpaths))
		for _, value := range slices.Sorted(maps.Keys(xpaths)) {
			fmt.Fprintln(&output, value)
		}
	}
	if showURLs {
		if showXPaths {
			output.WriteByte('\n')
		}
		fmt.Fprintf(&output, "URLs: %d\n", len(urls))
		for _, value := range slices.Sorted(maps.Keys(urls)) {
			fmt.Fprintln(&output, value)
		}
	}
	return output.String()
}

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

		targets := findLinkTargets(de.URLTarget)
		foundDataitem := false
		for _, target := range targets {
			switch target.kind {
			case "dataitem":
				foundDataitem = true
				processDataitemTarget(target.value)
			case "wwwpage":
				urls[target.value]++
			}
		}
		if strings.HasPrefix(de.URLTarget, "geneos://") {
			urls[de.URLTarget]++
		}
		if !foundDataitem {
			log.Debug("No dataitem matches found, skipping line", slog.String("URLTarget", de.URLTarget))
		}
	case dataItemSeverityNodeClass:
		var ds dataItemSeverityNodeFields
		if err := unmarshalLineToStruct(line, &ds); err != nil {
			fmt.Println("Error parsing DataItemSeverityNode line:", err)
			return
		}
		targets := findLinkTargets(ds.URLTarget)
		foundDataitem := false
		for _, target := range targets {
			switch target.kind {
			case "dataitem":
				foundDataitem = true
				processDataitemTarget(target.value)
			case "wwwpage":
				urls[target.value]++
			}
		}
		if strings.HasPrefix(ds.URLTarget, "geneos://") {
			urls[ds.URLTarget]++
		}
		if !foundDataitem {
			log.Debug("No dataitem matches found, skipping line", slog.String("URLTarget", ds.URLTarget))
		}
	case geneosURLClass:
		var gu geneosURL
		if err := unmarshalLineToStruct(line, &gu); err != nil {
			log.Debug("Error parsing GeneosURL line, skipping", slog.Any("error", err))
			return
		}

		targets := findLinkTargets(gu.URLTarget)
		if len(targets) == 1 && targets[0].start == 0 && targets[0].end == len(gu.URLTarget) {
			target := targets[0]
			links[target.kind+"(...)"]++
			if target.kind == "dataitem" {
				processDataitemTarget(target.value)
			} else {
				urls[target.value]++
			}
		} else if strings.HasPrefix(gu.URLTarget, "geneos://") {
			links["geneos://..."]++
			urls[gu.URLTarget]++
		} else {
			links["other"]++
			urls[gu.URLTarget]++
		}
	default:
		// something else, ignore
	}
}

func processDataitemTarget(di string) {
	xpaths[di]++
	xp, err := xpath.Parse(di)
	if err != nil {
		log.Debug("Failed to parse DataitemXpath, skipping", slog.Any("error", err), slog.String("DataitemXpath", di))
		return
	}
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
						types[*xp.Sampler.Type]++
					}
					if xp.Dataview != nil {
						dataviews[xp.Dataview.Name]++
					}
				}
			}
		}
	}
}
