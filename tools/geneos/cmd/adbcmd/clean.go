package adbcmd

import (
	"archive/zip"
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/geneos/xpath"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

//go:embed _docs/clean.md
var cleanCmdDescription string

var cleanKeepAll, cleanCmdKeepStartupData bool
var cleanOptions cleaningOptions
var cleanFilter string

func init() {
	adbCmd.AddCommand(cleanCmd)

	// xpath component flags
	cleanOptions = cleaningOptions{
		Gateways:   optionSlice{"remove-all"},
		Probes:     optionSlice{"remove-all"},
		Entities:   optionSlice{"remove-all"},
		Attributes: optionSlice{},
		Samplers:   optionSlice{"keep"},
		Types:      optionSlice{"keep"},
		Dataviews:  optionSlice{"keep"},
	}

	cleanCmd.Flags().BoolVarP(&cleanKeepAll, "keep-all", "k", false, "Keep all data not removed by other options (overrides defaults)")

	cleanCmd.Flags().Var(&cleanOptions.Gateways, "gateway", "Gateway name updates. Can be repeated, applied in order given. Valid options:\n'remove-all' | 'keep' | 'remove:pattern' | 'set:ABC' | 'replace:/<pattern>/<replacement>/'")

	cleanCmd.Flags().Var(&cleanOptions.Probes, "probe", "Probe name updates. Can be repeated, applied in order given. Valid options:\n'remove-all' | 'keep' | 'remove:pattern' | 'set:ABC' | 'replace:/<pattern>/<replacement>/'")

	cleanCmd.Flags().Var(&cleanOptions.Entities, "entity", "Entity name updates. Can be repeated, applied in order given. Valid options:\n'remove-all' | 'keep' | 'remove:pattern' | 'set:ABC' | 'replace:/<pattern>/<replacement>/'")

	cleanCmd.Flags().Var(&cleanOptions.Attributes, "attribute", "Attribute updates. Can be repeated, applied in order given. Valid options:\n'remove:KEY[=VALUE]' | 'add:KEY=VALUE' | 'replace:KEY=VALUE' | 'remove-all'")

	cleanCmd.Flags().Var(&cleanOptions.Samplers, "sampler", "Sampler name updates. Can be repeated, applied in order given. Valid options:\n'remove-all' | 'keep' | 'remove:pattern' | 'set:ABC' | 'replace:/<pattern>/<replacement>/'")

	cleanCmd.Flags().Var(&cleanOptions.Types, "type", "Type name updates. Can be repeated, applied in order given. Valid options:\n'remove-all' | 'keep' | 'remove:pattern' | 'set:ABC' | 'replace:/<pattern>/<replacement>/'")

	cleanCmd.Flags().Var(&cleanOptions.Dataviews, "dataview", "Dataview name updates. Can be repeated, applied in order given. Valid options:\n'remove-all' | 'keep' | 'remove:pattern' | 'set:ABC' | 'replace:/<pattern>/<replacement>/'")

	// TODO: colour schemes for severities - later

	// --severity-critical-colour R,G,B (Int8Slice)
	// --severity-warning-colour R,G,B (Int8Slice)
	// --severity-ok-colour R,G,B (Int8Slice)
	// --severity-undefined-colour R,G,B (Int8Slice)

	cleanCmd.Flags().StringVar(&cleanFilter, "filter", "", "Filter for cleaning specific XPaths (applies glob patterns)")

	cleanCmd.Flags().BoolVar(&cleanCmdKeepStartupData, "keep-startup-data", false, "Keep startup data (default is to remove)")

	cleanCmd.Flags().SortFlags = false
}

var cleanFilterXPath *xpath.XPath

var cleanCmd = &cobra.Command{
	Use:          "clean [flags] [PATH...]",
	Short:        "Clean ADB file(s)",
	Long:         cleanCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
		cmd.CmdAllowRoot:   "true",
	},
	RunE: func(command *cobra.Command, paths []string) (err error) {
		log = logger.Logger.With("command", "adb clean")
		if len(paths) == 0 {
			return fmt.Errorf("no dashboard files given on command line")
		}

		if cleanKeepAll {
			cleanOptions.Gateways[0] = "keep"
			cleanOptions.Probes[0] = "keep"
			cleanOptions.Entities[0] = "keep"
		}

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

		if cleanFilter != "" {
			cleanFilterXPath, err = xpath.Parse(cleanFilter)
			if err != nil {
				return fmt.Errorf("invalid filter XPath: %w", err)
			}
		}

		for _, file := range paths {
			if err = cleanCommand(file); err != nil {
				log.Error("Failed to clean file", "file", file, "error", err)
			}
		}

		return nil
	},
}

// cleanCommand processes a single Activedashboard file, creating a
// cleaned version with updated XPaths and optionally keeping startup
// data (any `.dbo` files in the archive).
func cleanCommand(file string) (err error) {
	zr, err := zip.OpenReader(file)
	if err != nil {
		return
	}
	defer zr.Close()

	w, err := os.Create(file + ".cleaned")
	if err != nil {
		return
	}
	// must close before remove on Windows, so put the remove on the
	// defer stack first
	defer os.Remove(file + ".cleaned")
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

				return err
			}
			wr, err := zw.Create(f.Name)
			if err != nil {
				return err
			}
			err = cleanActivedashboard(r, wr)
			if err != nil {
				return err
			}
		case ".dbo":
			// skip if told to, else copy as-is
			if cleanCmdKeepStartupData {
				err = zw.Copy(f)
				if err != nil {
					return
				}
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

	ts := time.Now().Format("20060102150405")
	os.Rename(file, file+"."+ts+".bak")
	os.Rename(file+".cleaned", file)
	return
}

// cleanActivedashboard processes an Activedashboard extension file,
// cleaning up DataItemExprNode and DataItemSeverityNode lines as
// specified by the command flags.
func cleanActivedashboard(r io.Reader, w io.Writer) (err error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		// both these classes have the URLTarget in field 4
		case strings.HasPrefix(line, dataItemExprNodeClass), strings.HasPrefix(line, dataItemSeverityNodeClass):
			// split into fields, manually for now - we can't marshal
			// yet, so just split and reassemble
			parts := strings.Split(line, string(fieldSeparator))

			// parts[4] may have multiple dataitem() values
			dataitems := strings.Split(parts[4], "dataitem(")
			if len(dataitems) > 1 {
				// throw away the first empty element resulting from the split
				dataitems = dataitems[1:]
				for i, di := range dataitems {
					dataitems[i] = strings.TrimSuffix(di, ")")
				}
			}

			for i, xp := range dataitems {
				xpp, err := xpath.Parse(xp)
				if err != nil {
					log.Error("Failed to parse DataitemXpath", slog.Any("error", err), slog.String("DataitemXpath", xp))
				}

				// filter here
				if cleanFilterXPath != nil && !xpp.Match(cleanFilterXPath) {
					log.Debug("Skipping DataitemXpath due to filter", slog.String("DataitemXpath", xpp.String()), slog.String("Filter", cleanFilterXPath.String()))
					dataitems[i] = "dataitem(" + xp + ")"
					continue
				}

				if xpp.Gateway != nil {
					xpp.Gateway.Name = updateNameFromOptions(xpp.Gateway.Name, cleanOptions.Gateways)
				} else if len(cleanOptions.Gateways) > 0 {
					name := updateNameFromOptions("", cleanOptions.Gateways)
					if name != "" {
						xpp.Gateway = &xpath.Gateway{Name: name}
					}
				}

				if xpp.Probe != nil {
					xpp.Probe.Name = updateNameFromOptions(xpp.Probe.Name, cleanOptions.Probes)
				} else if len(cleanOptions.Probes) > 0 {
					name := updateNameFromOptions("", cleanOptions.Probes)
					if name != "" {
						xpp.Probe = &xpath.Probe{Name: name}
					}
				}

				if xpp.Entity != nil {
					xpp.Entity.Name = updateNameFromOptions(xpp.Entity.Name, cleanOptions.Entities)
				} else if len(cleanOptions.Entities) > 0 {
					name := updateNameFromOptions("", cleanOptions.Entities)
					if name != "" {
						xpp.Entity = &xpath.Entity{Name: name}
					}
				}

				// update attributes, checking if entity exists etc.
				if len(cleanOptions.Attributes) > 0 {
					var attr map[string]string
					if xpp.Entity != nil {
						attr = xpp.Entity.Attributes
					}
					attr = updateAttributesFromOptions(attr, cleanOptions.Attributes)
					if xpp.Entity != nil {
						// override, even if resulting map is empty
						xpp.Entity.Attributes = attr
					} else {
						if len(attr) > 0 {
							// entity does not exist, create it with the updated attributes
							xpp.Entity = &xpath.Entity{
								Attributes: attr,
							}
						}
					}
				}

				// update sampler name first, if required - types below
				if xpp.Sampler != nil {
					xpp.Sampler.Name = updateNameFromOptions(xpp.Sampler.Name, cleanOptions.Samplers)
				} else if len(cleanOptions.Samplers) > 0 {
					xpp.Sampler = &xpath.Sampler{
						Name: updateNameFromOptions("", cleanOptions.Samplers),
					}
				}

				// updating the type name is more complicated
				var typ *string
				if len(cleanOptions.Types) > 0 {
					if xpp.Sampler != nil && xpp.Sampler.Type != nil {
						t := updateNameFromOptions(*xpp.Sampler.Type, cleanOptions.Types)
						typ = &t
					} else {
						t := updateNameFromOptions("", cleanOptions.Types)
						typ = &t
					}
				}

				if typ != nil {
					if xpp.Sampler != nil {
						if *typ == "" {
							// if result is an empty string then remove it
							xpp.Sampler.Type = nil
						} else {
							xpp.Sampler.Type = typ
						}
					} else if *typ != "" {
						// only create a new sampler if the type is not
						// empty
						xpp.Sampler = &xpath.Sampler{
							Type: typ,
						}
					}
				}

				if xpp.Dataview != nil {
					xpp.Dataview.Name = updateNameFromOptions(xpp.Dataview.Name, cleanOptions.Dataviews)
				} else if len(cleanOptions.Dataviews) > 0 {
					name := updateNameFromOptions("", cleanOptions.Dataviews)
					if name != "" {
						xpp.Dataview = &xpath.Dataview{Name: name}
					}
				}

				dataitems[i] = "dataitem(" + xpp.String() + ")"
			}

			parts[4] = strings.Join(dataitems, "")
			parts[5] = EMPTY_STRING

			// join it back up
			line = strings.Join(parts, string(fieldSeparator))

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

// updateNameFromOptions updates the given name based on the provided
// options. It supports options "remove-all", "remove:pattern", "keep", "set:ABC",
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
			re, err := regexp.Compile(pattern)
			if err == nil {
				name = re.ReplaceAllString(name, "")
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

// updateAttributesFromOptions updates the given attribute map based on
// the provided options. It supports options:
//
//	"remove-all" - remove all attributes by clearing the attribute map
//	"remove:KEY[=VALUE]" - remove the attribute with the specified KEY and optional VALUE. If VALUE is provided, only remove the attribute if it matches the given VALUE.
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
				if attr[key] == value {
					delete(attr, key)
				}
			} else {
				delete(attr, key)
			}
		case strings.HasPrefix(opt, "add:"):
			key, value, found := strings.Cut(strings.TrimPrefix(opt, "add:"), "=")
			if found {
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
}

func (o *optionSlice) String() string {
	return fmt.Sprintf("%v", *o)
}

func (o *optionSlice) Set(value string) error {
	*o = append(*o, value)
	return nil
}

func (o *optionSlice) Type() string {
	return "OPTION"
}
