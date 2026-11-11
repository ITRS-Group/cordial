package adbcmd

import (
	"archive/zip"
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/pkg/geneos/xpath"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

//go:embed _docs/clean.md
var cleanCmdDescription string

var cleanCmdKeepAll, cleanCmdKeepStartupData, cleanCmdSkipModifiers, cleanCmdSkipLinks bool
var cleanOptions cleaningOptions
var cleanCmdOutput string

type filterType struct {
	XPath      string
	Gateway    string
	Probe      string
	Entity     string
	Attributes optionSlice
	Sampler    string
	Type       string
	Dataview   string
	URLs       optionSlice
}

var filters filterType = filterType{}
var cleanURLFilters []*regexp.Regexp

func init() {
	adbCmd.AddCommand(cleanCmd)

	// xpath component flags
	cleanOptions = cleaningOptions{
		Gateways:   optionSlice{"remove-all"},
		Probes:     optionSlice{"remove-all"},
		Entities:   optionSlice{"remove-all"},
		Attributes: optionSlice{},
		Samplers:   optionSlice{"keep"},
		Types:      optionSlice{"remove-empty"},
		Dataviews:  optionSlice{"keep"},
		WebPages:   optionSlice{},
	}

	cleanCmd.Flags().StringVarP(&cleanCmdOutput, "output", "o", "",

		`Specify the output `+"`directory`"+` for cleaned ADB files.
If not specified, the cleaned files will replace the
original files in place and a backup of the original
will be created with a timestamp added before the ".adb"
extension.
`)

	cleanCmd.Flags().Var(&cleanOptions.Gateways, "gateway", "Gateway name updates. Default 'remove-all'")

	cleanCmd.Flags().Var(&cleanOptions.Probes, "probe", "Probe name updates. Default 'remove-all'")

	cleanCmd.Flags().Var(&cleanOptions.Entities, "entity", "Entity name updates. Default 'remove-all'")

	cleanCmd.Flags().Var(&cleanOptions.Samplers, "sampler", "Sampler name updates. Default 'keep'")

	cleanCmd.Flags().Var(&cleanOptions.Dataviews, "dataview",
		`Dataview name updates. Default 'keep'.

For the above xpath components the valid options, which can be
repeated and are applied in the order given, are:

'remove-all'          - remove all names
'keep'                - keep all names
'remove:NAME'         - remove names matching NAME. NAME can be
                        a glob (not regexp) pattern
'set:ABC'             - set the name to ABC
'replace:/P/R/'       - apply replacement R to names
                        matching the regexp pattern R. The
                        replacement is applied as many times as
                        P matches
`,
	)
	cleanCmd.Flags().Var(&cleanOptions.WebPages, "wwwpage",
		`Web page URL updates. Apply regular-expression replacements to the URL
inside each wwwpage(...) target. Repeat to apply actions in order:

'replace:#PATTERN#REPLACEMENT#' - replace matching URL text. The delimiter
can be any character and must not conflict with the pattern or replacement.
`,
	)

	cleanCmd.Flags().Var(&cleanOptions.Types, "type",

		`Type updates. Default 'remove-empty'.

As for xpath components above, but with the addtional options:

'remove-empty'        - Remove types with an empty name
'set-empty'           - Set types with an empty name. Note that
                        this is different from no type.
`)

	cleanCmd.Flags().Var(&cleanOptions.Attributes, "attribute",
		`Attribute updates.

For each attribute, the valid options, which can be repeated and
are applied in the order given, are:

'remove:NAME[=VALUE]' - remove attribute NAME, optionally only
                        those matching VALUE. Both KEY and VALUE
                        can be glob (not regexp) patterns
'add:NAME=VALUE'      - add attribute NAME with VALUE
'replace:NAME=VALUE'  - replace attribute NAME with VALUE
'replace:NAME=/P/R/'  - replace attribute NAME with
                        the result of applying the regexp
                        pattern P to replacement R. The
                        replacement is applied as many times as
                        P matches
'remove-all'          - remove all attributes
`)

	// TODO: colour schemes for severities - later

	// --severity-critical-colour R,G,B (Int8Slice)
	// --severity-warning-colour R,G,B (Int8Slice)
	// --severity-ok-colour R,G,B (Int8Slice)
	// --severity-undefined-colour R,G,B (Int8Slice)

	cleanCmd.Flags().StringVar(&filters.XPath, "filter-xpath", "", "Filter for matching XPaths, leaving others unchanged\n(applies glob patterns not xpath expressions, e.g. '//managedEntity[(@name=\"LDN*\")]')")
	cleanCmd.Flags().StringVar(&filters.Gateway, "filter-gateway", "", "Filter for matching gateways")
	cleanCmd.Flags().StringVar(&filters.Probe, "filter-probe", "", "Filter for matching probes")
	cleanCmd.Flags().StringVar(&filters.Entity, "filter-entity", "", "Filter for matching entities")
	cleanCmd.Flags().StringVar(&filters.Sampler, "filter-sampler", "", "Filter for matching samplers")
	cleanCmd.Flags().StringVar(&filters.Type, "filter-type", "", "Filter for matching types")
	cleanCmd.Flags().StringVar(&filters.Dataview, "filter-dataview", "",
		`Filter for matching dataviews

All the above filters support glob patterns (not regular expressions).
`,
	)
	cleanCmd.Flags().Var(&filters.Attributes, "filter-attribute", "Filter for matching attributes in the form `NAME=VALUE` where\nVALUE should be '*' if you want to match any value. Can be repeated.\n")
	cleanCmd.Flags().Var(&filters.URLs, "filter-url", "Only process wwwpage URLs matching any supplied regular expression. Can be repeated.\n")

	cleanCmd.Flags().BoolVarP(&cleanCmdKeepAll, "keep-all", "k", false, "Set default action for all processing actions to 'keep' or do nothing")
	cleanCmd.Flags().BoolVar(&cleanCmdSkipModifiers, "skip-modifiers", false, "Skip processing of modifier elements")
	cleanCmd.Flags().BoolVar(&cleanCmdSkipLinks, "skip-links", false, "Skip processing of link elements")
	cleanCmd.Flags().BoolVar(&cleanCmdKeepStartupData, "keep-startup-data", false, "Keep startup data (default is to remove)")

	// don't show default values
	cleanCmd.Flags().Lookup("gateway").DefValue = ""
	cleanCmd.Flags().Lookup("probe").DefValue = ""
	cleanCmd.Flags().Lookup("entity").DefValue = ""
	cleanCmd.Flags().Lookup("attribute").DefValue = ""
	cleanCmd.Flags().Lookup("sampler").DefValue = ""
	cleanCmd.Flags().Lookup("type").DefValue = ""
	cleanCmd.Flags().Lookup("dataview").DefValue = ""
	cleanCmd.Flags().Lookup("filter-attribute").DefValue = ""
	cleanCmd.Flags().Lookup("wwwpage").DefValue = ""
	cleanCmd.Flags().Lookup("filter-url").DefValue = ""

	// add aliases for flags if needed
	cleanCmd.Flags().SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		switch name {
		case "gateways":
			name = "gateway"
		case "probes":
			name = "probe"
		case "entities":
			name = "entity"
		case "attributes":
			name = "attribute"
		case "samplers":
			name = "sampler"
		case "types":
			name = "type"
		case "dataviews":
			name = "dataview"
		}
		return pflag.NormalizedName(name)
	})

	cleanCmd.Flags().SortFlags = false
}

var cleanFilterXPath *xpath.XPath = &xpath.XPath{}

func applyCleanDefaults(editMode, startupDataExplicit bool) {
	if cleanCmdKeepAll || editMode {
		cleanOptions.Gateways[0] = "keep"
		cleanOptions.Probes[0] = "keep"
		cleanOptions.Entities[0] = "keep"
	}
	if editMode {
		if len(cleanOptions.Types) == 1 && cleanOptions.Types[0] == "remove-empty" {
			cleanOptions.Types[0] = "keep"
		}
		if !startupDataExplicit {
			cleanCmdKeepStartupData = true
		}
	}
}

var cleanCmd = &cobra.Command{
	Use:          "clean [flags] [PATH...]",
	Aliases:      []string{"edit"},
	Short:        "Clean or edit ADB file(s)",
	Long:         cleanCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:       "false",
		cmd.CmdRequireHome:  "false",
		cmd.CmdAllowRoot:    "true",
		cmd.CmdAuditCommand: "false", // no environmental effect
		cmd.CmdAuditActions: "false",
	},
	RunE: func(command *cobra.Command, paths []string) (err error) {
		log = logger.Logger.With("command", "adb clean")
		if len(paths) == 0 {
			return fmt.Errorf("no dashboard files given on command line")
		}

		if cleanCmdSkipModifiers && cleanCmdSkipLinks {
			return fmt.Errorf("cannot skip both modifiers and links")
		}

		applyCleanDefaults(command.CalledAs() == "edit", command.Flags().Changed("keep-startup-data"))

		// update options elements to remove initial default if any
		// other options have been set
		if len(cleanOptions.Gateways) > 1 {
			cleanOptions.Gateways = cleanOptions.Gateways[1:]
		}
		if len(cleanOptions.Probes) > 1 {
			cleanOptions.Probes = cleanOptions.Probes[1:]
		}
		if len(cleanOptions.Entities) > 1 {
			cleanOptions.Entities = cleanOptions.Entities[1:]
		}
		if len(cleanOptions.Samplers) > 1 {
			cleanOptions.Samplers = cleanOptions.Samplers[1:]
		}
		if len(cleanOptions.Types) > 1 {
			cleanOptions.Types = cleanOptions.Types[1:]
		}
		if len(cleanOptions.Dataviews) > 1 {
			cleanOptions.Dataviews = cleanOptions.Dataviews[1:]
		}

		if filters.XPath != "" {
			cleanFilterXPath, err = xpath.Parse(filters.XPath)
			if err != nil {
				return fmt.Errorf("invalid filter XPath: %w", err)
			}
		}
		cleanURLFilters = make([]*regexp.Regexp, 0, len(filters.URLs))
		for _, pattern := range filters.URLs {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("invalid URL filter %q: %w", pattern, err)
			}
			cleanURLFilters = append(cleanURLFilters, re)
		}

		// these have to be evaluated in order as ResolveTo() will
		// remove items to the right of the element passed

		if filters.Gateway != "" {
			cleanFilterXPath = cleanFilterXPath.ResolveTo(&xpath.Gateway{Name: filters.Gateway})
		}

		if filters.Probe != "" {
			cleanFilterXPath = cleanFilterXPath.ResolveTo(&xpath.Probe{Name: filters.Probe})
		}

		if filters.Entity != "" {
			cleanFilterXPath = cleanFilterXPath.ResolveTo(&xpath.Entity{Name: filters.Entity})
		}

		if len(filters.Attributes) > 0 {
			if cleanFilterXPath.Entity == nil {
				cleanFilterXPath = cleanFilterXPath.ResolveTo(&xpath.Entity{})
			}
			for _, a := range filters.Attributes {
				name, value, found := strings.Cut(a, "=")
				if !found {
					value = "*"
				}
				cleanFilterXPath.Entity.Attributes[name] = value
			}
		}

		if filters.Sampler != "" {
			log.Debug("Setting Sampler filter", slog.String("Sampler", filters.Sampler))
			cleanFilterXPath = cleanFilterXPath.ResolveTo(&xpath.Sampler{Name: filters.Sampler})
		}

		if filters.Type != "" {
			cleanFilterXPath = cleanFilterXPath.ResolveTo(&xpath.Sampler{})
			cleanFilterXPath.Sampler.Type = &filters.Type
		}

		if filters.Dataview != "" {
			cleanFilterXPath = cleanFilterXPath.ResolveTo(&xpath.Dataview{Name: filters.Dataview})
		}

		for _, file := range paths {
			var stats cleanStats
			if stats, err = cleanCommand(file, cleanCmdOutput); err != nil {
				log.Error("Failed to clean file", "file", file, "error", err)
			} else {
				fmt.Printf("%s: %s\n", file, stats.summary())
			}
		}

		return nil
	},
}

// cleanCommand processes a single Activedashboard file, creating a
// cleaned version with updated XPaths and optionally keeping startup
// data (any `.dbo` files in the archive).
type cleanStats struct {
	xpathLinks             int
	urlLinks               int
	modifierAliasesCleared int
	startupDataRemoved     int
	gatewaysChanged        int
	probesChanged          int
	entitiesChanged        int
	attributesChanged      int
	samplersChanged        int
	typesChanged           int
	dataviewsChanged       int
}

func (s *cleanStats) recordXPathChange(before, after *xpath.XPath) {
	s.xpathLinks++
	if (before.Gateway == nil) != (after.Gateway == nil) || before.Gateway != nil && before.Gateway.Name != after.Gateway.Name {
		s.gatewaysChanged++
	}
	if (before.Probe == nil) != (after.Probe == nil) || before.Probe != nil && before.Probe.Name != after.Probe.Name {
		s.probesChanged++
	}
	if (before.Entity == nil) != (after.Entity == nil) || before.Entity != nil && before.Entity.Name != after.Entity.Name {
		s.entitiesChanged++
	}
	var beforeAttributes, afterAttributes map[string]string
	if before.Entity != nil {
		beforeAttributes = before.Entity.Attributes
	}
	if after.Entity != nil {
		afterAttributes = after.Entity.Attributes
	}
	if !maps.Equal(beforeAttributes, afterAttributes) {
		s.attributesChanged++
	}
	if (before.Sampler == nil) != (after.Sampler == nil) || before.Sampler != nil && before.Sampler.Name != after.Sampler.Name {
		s.samplersChanged++
	}
	var beforeType, afterType *string
	if before.Sampler != nil {
		beforeType = before.Sampler.Type
	}
	if after.Sampler != nil {
		afterType = after.Sampler.Type
	}
	if !equalOptionalString(beforeType, afterType) {
		s.typesChanged++
	}
	if (before.Dataview == nil) != (after.Dataview == nil) || before.Dataview != nil && before.Dataview.Name != after.Dataview.Name {
		s.dataviewsChanged++
	}
}

func equalOptionalString(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func (s cleanStats) summary() string {
	linkCount := s.xpathLinks + s.urlLinks
	parts := make([]string, 0, 4)
	if linkCount == 0 {
		parts = append(parts, "No links updated")
	} else {
		linkTypes := make([]string, 0, 2)
		if s.xpathLinks > 0 {
			linkTypes = append(linkTypes, pluralCount(s.xpathLinks, "XPath"))
		}
		if s.urlLinks > 0 {
			linkTypes = append(linkTypes, pluralCount(s.urlLinks, "URL"))
		}
		parts = append(parts, fmt.Sprintf("%s updated (%s)", pluralCount(linkCount, "link"), strings.Join(linkTypes, ", ")))
	}

	xpathChanges := make([]string, 0, 7)
	for _, change := range []struct {
		name  string
		count int
	}{
		{"Gateway", s.gatewaysChanged},
		{"Probe", s.probesChanged},
		{"Managed Entity", s.entitiesChanged},
		{"Attribute", s.attributesChanged},
		{"Sampler", s.samplersChanged},
		{"Type", s.typesChanged},
		{"Dataview", s.dataviewsChanged},
	} {
		if change.count > 0 {
			xpathChanges = append(xpathChanges, pluralCount(change.count, change.name))
		}
	}
	if len(xpathChanges) > 0 {
		parts = append(parts, "XPath changes: "+strings.Join(xpathChanges, ", "))
	}
	if s.modifierAliasesCleared > 0 {
		parts = append(parts, pluralCount(s.modifierAliasesCleared, "modifier alias")+" cleared")
	}
	if s.startupDataRemoved > 0 {
		parts = append(parts, pluralCount(s.startupDataRemoved, "startup-data entry")+" removed")
	}
	return strings.Join(parts, "; ")
}

func pluralCount(count int, singular string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	if strings.HasSuffix(singular, "entry") || strings.HasSuffix(singular, "entity") {
		return fmt.Sprintf("%d %sies", count, strings.TrimSuffix(singular, "y"))
	}
	if strings.HasSuffix(singular, "alias") {
		return fmt.Sprintf("%d %ses", count, singular)
	}
	return fmt.Sprintf("%d %ss", count, singular)
}

func cleanCommand(file, outputDir string) (stats cleanStats, err error) {
	dest := file

	// create output directory if specified, or fail
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0775); err != nil {
			return stats, err
		}
		dest = filepath.Join(outputDir, filepath.Base(file))
	}

	zr, err := zip.OpenReader(file)
	if err != nil {
		return
	}
	defer zr.Close()

	w, err := os.Create(dest + ".cleaned")
	if err != nil {
		return
	}
	// must close before remove on Windows, so put the remove on the
	// defer stack first
	defer os.Remove(dest + ".cleaned")
	defer w.Close()

	zw := zip.NewWriter(w)
	defer zw.Close()

	// Look for the Activedashboard extension file
	for _, f := range zr.File {
		switch path.Ext(f.Name) {
		case extension:
			// do stuff and write it to the archive
			r, err := f.Open()
			if err != nil {
				return stats, err
			}
			wr, err := zw.Create(f.Name)
			if err != nil {
				return stats, err
			}
			err = cleanActivedashboardWithStats(r, wr, &stats)
			if err != nil {
				return stats, err
			}
		case ".dbo":
			// skip if told to, else copy as-is
			if cleanCmdKeepStartupData {
				err = zw.Copy(f)
				if err != nil {
					return
				}
			} else {
				stats.startupDataRemoved++
			}
		default:
			// copy everything else as-is
			err = zw.Copy(f)
			if err != nil {
				return
			}
		}
	}

	// rename file

	// close handles for windows before renames
	zw.Close()
	w.Close()
	zr.Close()

	backup := true
	if outputDir != "" {
		absOutDir, err := filepath.Abs(outputDir)
		if err != nil {
			return stats, err
		}
		srcDir, err := filepath.Abs(filepath.Dir(file))
		if err != nil {
			return stats, err
		}

		// if the source and dest directories are different, no backup is needed
		if absOutDir != srcDir {
			backup = false
		}
	}

	if backup {
		// back up original file with timestamp
		ts := time.Now().Format("20060102150405")
		ext := filepath.Ext(file)
		pre := strings.TrimSuffix(file, ext)
		if err = os.Rename(file, pre+"-"+ts+ext); err != nil {
			// fail if we cannot backup file
			return
		}
	}

	err = os.Rename(dest+".cleaned", dest)
	return stats, err
}

// cleanActivedashboard processes an Activedashboard extension file,
// cleaning up DataItemExprNode and DataItemSeverityNode lines as
// specified by the command flags.
func cleanActivedashboard(r io.Reader, w io.Writer) (err error) {
	return cleanActivedashboardWithStats(r, w, nil)
}

func cleanActivedashboardWithStats(r io.Reader, w io.Writer, stats *cleanStats) (err error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		// both these classes have the URLTarget in field 4
		case strings.HasPrefix(line, dataItemExprNodeClass), strings.HasPrefix(line, dataItemSeverityNodeClass):
			// skip if told to, else process normally
			if cleanCmdSkipModifiers {
				break
			}

			// split into fields, manually for now - we can't re-marshal
			// yet, so just split and reassemble rather than unmarshal
			// like info does
			parts := strings.Split(line, string(fieldSeparator))

			modifierChanged := false
			parts[4] = rewriteLinkTargets(parts[4], func(kind, target string) string {
				switch kind {
				case "dataitem":
					xpp, err := xpath.Parse(target)
					if err != nil {
						log.Warn("Failed to parse DataitemXpath, skipping", slog.Any("error", err), slog.String("DataitemXpath", target))
						return target
					}
					if cleanFilterXPath != nil && !xpp.Match(cleanFilterXPath) {
						log.Debug("Skipping DataitemXpath due to filter", slog.String("DataitemXpath", xpp.String()), slog.String("Filter", cleanFilterXPath.String()))
						return target
					}
					originalPath := xpp.String()
					updatedPath := updateDataitemXpath(xpp, cleanOptions).String()
					if updatedPath != originalPath {
						modifierChanged = true
						if stats != nil {
							originalXPath, _ := xpath.Parse(originalPath)
							stats.recordXPathChange(originalXPath, xpp)
						}
					}
					return updatedPath
				case "wwwpage":
					updatedURL := updateWebPageURL(target, cleanOptions.WebPages, cleanURLFilters)
					if updatedURL != target && stats != nil {
						stats.urlLinks++
					}
					return updatedURL
				default:
					return target
				}
			})
			if modifierChanged {
				if parts[5] != EMPTY_STRING && stats != nil {
					stats.modifierAliasesCleared++
				}
				parts[5] = EMPTY_STRING
			}

			// join it back up
			line = strings.Join(parts, string(fieldSeparator))

		case strings.HasPrefix(line, geneosURLClass):
			var gu geneosURL

			// skip if told to, else process normally
			if cleanCmdSkipLinks {
				break
			}
			if err := unmarshalLineToStruct(line, &gu); err != nil {
				fmt.Println("Error parsing GeneosURL line:", err)
				break
			}
			targets := findLinkTargets(gu.URLTarget)
			if len(targets) != 1 || targets[0].start != 0 || targets[0].end != len(gu.URLTarget) {
				// GeneosURL records contain one complete target per entry.
				break
			}
			target := targets[0]
			updatedTarget := target.value
			switch target.kind {
			case "dataitem":
				xp, err := xpath.Parse(target.value)
				if err != nil {
					log.Warn("Failed to parse DataitemXpath, skipping", slog.Any("error", err), slog.String("DataitemXpath", gu.URLTarget))
					break
				}
				if cleanFilterXPath != nil && !xp.Match(cleanFilterXPath) {
					log.Debug("Skipping DataitemXpath due to filter", slog.String("DataitemXpath", xp.String()), slog.String("Filter", cleanFilterXPath.String()))
					break
				}
				updatedTarget = updateDataitemXpath(xp, cleanOptions).String()
				if updatedTarget != target.value && stats != nil {
					originalXPath, _ := xpath.Parse(target.value)
					stats.recordXPathChange(originalXPath, xp)
				}
			case "wwwpage":
				updatedTarget = updateWebPageURL(target.value, cleanOptions.WebPages, cleanURLFilters)
				if updatedTarget != target.value && stats != nil {
					stats.urlLinks++
				}
			}
			if updatedTarget != target.value {
				parts := strings.Split(line, string(fieldSeparator))
				parts[2] = target.kind + "(" + updatedTarget + ")"
				line = strings.Join(parts, string(fieldSeparator))
			}
		default:
			// skip all other lines, leave them as-is
		}

		// write it out
		_, err = fmt.Fprintln(w, line)
		if err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading Activedashboard file: %v", err)
	}
	return
}

func updateDataitemXpath(xpp *xpath.XPath, opts cleaningOptions) *xpath.XPath {
	if xpp.Gateway != nil {
		xpp.Gateway.Name = updateNameFromOptions(xpp.Gateway.Name, opts.Gateways)
	} else if len(opts.Gateways) > 0 {
		name := updateNameFromOptions("", opts.Gateways)
		if name != "" {
			xpp.Gateway = &xpath.Gateway{Name: name}
		}
	}

	if xpp.Probe != nil {
		xpp.Probe.Name = updateNameFromOptions(xpp.Probe.Name, opts.Probes)
	} else if len(opts.Probes) > 0 {
		name := updateNameFromOptions("", opts.Probes)
		if name != "" {
			xpp.Probe = &xpath.Probe{Name: name}
		}
	}

	if xpp.Entity != nil {
		xpp.Entity.Name = updateNameFromOptions(xpp.Entity.Name, opts.Entities)
	} else if len(opts.Entities) > 0 {
		name := updateNameFromOptions("", opts.Entities)
		if name != "" {
			xpp.Entity = &xpath.Entity{Name: name}
		}
	}

	// Attribute actions do not create a Managed Entity. Set the entity first.
	if len(opts.Attributes) > 0 && xpp.Entity != nil {
		xpp.Entity.Attributes = updateAttributesFromOptions(xpp.Entity.Attributes, opts.Attributes)
	}

	// update sampler name first, if required - types below
	if xpp.Sampler != nil {
		xpp.Sampler.Name = updateNameFromOptions(xpp.Sampler.Name, opts.Samplers)
	} else if len(opts.Samplers) > 0 {
		xpp.Sampler = &xpath.Sampler{
			Name: updateNameFromOptions("", opts.Samplers),
		}
	}

	// updating the type name is more complicated

	// first process it
	var typ *string
	if xpp.Sampler != nil {
		typ = xpp.Sampler.Type
	}
	if len(opts.Types) > 0 {
		typ = updateTypeNameFromOptions(typ, opts.Types)
	}

	// now apply the processed type to the sampler, but only if necessary
	if xpp.Sampler != nil {
		xpp.Sampler.Type = typ
	} else if typ != nil {
		// only create a new sampler if the type is not
		// empty
		xpp.Sampler = &xpath.Sampler{
			Type: typ,
		}
	}

	if xpp.Dataview != nil {
		xpp.Dataview.Name = updateNameFromOptions(xpp.Dataview.Name, opts.Dataviews)
	} else if len(opts.Dataviews) > 0 {
		name := updateNameFromOptions("", opts.Dataviews)
		if name != "" {
			xpp.Dataview = &xpath.Dataview{Name: name}
		}
	}
	return xpp
}

// updateNameFromOptions updates the given name based on the provided
// options. It supports options "remove-all", "remove:NAME", "keep", "set:ABC",
// and "replace:/<pattern>/<replacement>/". "keep" is the default
// behavior and just returns the existing value unchanged.
func updateNameFromOptions(name string, options optionSlice) string {
	// Implement the logic to update the name based on the provided options.
	// This is a placeholder implementation.
	for _, opt := range options {
		switch {
		case opt == "remove-all":
			name = ""
		case strings.HasPrefix(opt, "remove:"):
			pattern := strings.TrimPrefix(opt, "remove:")
			if match, _ := path.Match(pattern, name); match {
				name = ""
			}
		case strings.HasPrefix(opt, "set:"):
			name = strings.TrimPrefix(opt, "set:")
		case strings.HasPrefix(opt, "replace:"):
			spec := strings.TrimPrefix(opt, "replace:")
			name = replaceAll(name, spec)
		case opt == "keep":
			// do nothing, keep the current name
		default:
			// do nothing (keep)
		}
	}
	return name
}

// updateTypeNameFromOptions updates the given type name based on the
// provided options. It supports options:
//
//	"remove-all" - remove types from all entries
//	"remove-empty" - remove the type if the current name is an empty string
//	"remove:NAME" - remove the type if it matches the given NAME. NAME can be a glob (not regexp) pattern, e.g. "PROD*".
//	"keep" - do nothing
//	"set:ABC" - set the type to the specified value
//	"set-empty" - set the type to an empty string
//	"replace:/<pattern>/<replacement>/" - replace the type using the given replacement spec. note there is no way to remove the type for a specific pattern
//
// "keep" is the default behavior and just returns the existing value
// unchanged.
func updateTypeNameFromOptions(input *string, options optionSlice) (result *string) {
	if input != nil {
		name := *input
		result = &name
	}
	for _, opt := range options {
		switch {
		case opt == "remove-all":
			result = nil
		case opt == "remove-empty":
			if result != nil && *result == "" {
				result = nil
			}
		case strings.HasPrefix(opt, "remove:"):
			if result != nil {
				pattern := strings.TrimPrefix(opt, "remove:")
				if match, _ := path.Match(pattern, *result); match {
					result = nil
				}
			}
		case strings.HasPrefix(opt, "set:"):
			resultStr := strings.TrimPrefix(opt, "set:")
			result = &resultStr
		case opt == "set-empty":
			resultStr := ""
			result = &resultStr
		case strings.HasPrefix(opt, "replace:"):
			if result != nil {
				spec := strings.TrimPrefix(opt, "replace:")
				resultStr := replaceAll(*result, spec)
				result = &resultStr
			}
		case opt == "keep":
			// do nothing, keep the current name
		default:
			// do nothing (keep)
		}
	}
	return result
}

// updateAttributesFromOptions updates the given attribute map based on
// the provided options. It supports options:
//
//	"remove-all" - remove all attributes by clearing the attribute map
//	"remove:KEY[=VALUE]" - remove the attribute with KEY and optional VALUE. Both can be glob patterns. If VALUE is provided, only remove the attribute if it matches the given VALUE.
//	"add:KEY=VALUE" - set the attribute with the specified KEY to the given VALUE
//	"replace:KEY=VALUE" - replace the VALUE of the attribute KEY
//	"replace:KEY=/pattern/replacement/" - replace the value of the attribute KEY using the given replacement spec
//
// The options are applied in the order given.
func updateAttributesFromOptions(attr map[string]string, options optionSlice) map[string]string {
	for _, opt := range options {
		switch {
		case opt == "remove-all":
			attr = make(map[string]string)
		case strings.HasPrefix(opt, "remove:"):
			key, value, found := strings.Cut(strings.TrimPrefix(opt, "remove:"), "=")
			if found {
				maps.DeleteFunc(attr, func(k string, v string) bool {
					matchK, _ := path.Match(key, k)
					matchV, _ := path.Match(value, v)
					return matchK && matchV
				})
			} else {
				maps.DeleteFunc(attr, func(k string, _ string) bool {
					match, _ := path.Match(key, k)
					return match
				})
			}
		case strings.HasPrefix(opt, "add:"):
			key, value, found := strings.Cut(strings.TrimPrefix(opt, "add:"), "=")
			if found {
				if attr == nil {
					attr = make(map[string]string)
				}
				attr[key] = value
			}
		case strings.HasPrefix(opt, "replace:"):
			key, value, found := strings.Cut(strings.TrimPrefix(opt, "replace:"), "=")
			if found {
				if strings.HasPrefix(value, "/") {
					if oldValue, ok := attr[key]; ok {
						attr[key] = replaceAll(oldValue, value)
					}
				} else {
					if _, ok := attr[key]; ok {
						attr[key] = value
					}
				}
			}
		default:
		}
	}
	return attr
}

// replaceAll applies the replacement specified by spec to the given
// name. The spec should be in the format /<pattern>/<replacement>/ with
// a delimiter of your choice (e.g., / or #). Delimiters can be escaped
// with a backslash.
func replaceAll(name, spec string) string {
	if len(spec) < 2 {
		log.Error("Invalid replace option: missing delimiter", slog.String("pattern", spec))
		return name
	}
	delim := spec[0]
	if spec[len(spec)-1] != delim {
		log.Error("Invalid replace option: delimiter mismatch", slog.String("pattern", spec))
		return name
	}

	// Split at the first delimiter that is not escaped by a backslash.
	splitAt := -1
	for i := 1; i < len(spec)-1; i++ {
		if spec[i] == '\\' {
			i++
			continue
		}
		if spec[i] == delim {
			splitAt = i
			break
		}
	}
	if splitAt < 0 {
		log.Error("Invalid replace option: missing pattern/replacement separator", slog.String("pattern", spec))
		return name
	}

	unescapeDelimiter := func(value string) string {
		return strings.ReplaceAll(value, "\\"+string(delim), string(delim))
	}
	pattern := unescapeDelimiter(spec[1:splitAt])
	replacement := unescapeDelimiter(spec[splitAt+1 : len(spec)-1])
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Error("Invalid replace option pattern", slog.String("pattern", pattern), slog.Any("error", err))
		return name
	}

	return re.ReplaceAllString(name, replacement)
}

type optionSlice []string

type cleaningOptions struct {
	Gateways   optionSlice
	Probes     optionSlice
	Entities   optionSlice
	Attributes optionSlice
	Samplers   optionSlice
	Types      optionSlice
	Dataviews  optionSlice
	WebPages   optionSlice
}

func updateWebPageURL(url string, options optionSlice, filters []*regexp.Regexp) string {
	if len(filters) > 0 {
		matched := false
		for _, filter := range filters {
			if filter.MatchString(url) {
				matched = true
				break
			}
		}
		if !matched {
			return url
		}
	}
	for _, option := range options {
		if strings.HasPrefix(option, "replace:") {
			url = replaceAll(url, strings.TrimPrefix(option, "replace:"))
		}
	}
	return url
}

// String TODO: Document its behavior, inputs, and effects.
func (o *optionSlice) String() string {
	return fmt.Sprintf("%v", *o)
}

// Set TODO: Document its behavior, inputs, and effects.
func (o *optionSlice) Set(value string) error {
	*o = append(*o, value)
	return nil
}

// Type TODO: Document its behavior, inputs, and effects.
func (o *optionSlice) Type() string {
	return "OPTION"
}
