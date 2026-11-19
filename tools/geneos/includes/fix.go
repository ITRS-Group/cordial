package includes

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	geneosxml "github.com/itrs-group/cordial/pkg/geneos"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

//go:embed _docs/fix.md
var includeFixCmdDescription string

func init() {
	includesCmd.AddCommand(includesFixCmd)
	includesFixCmd.Flags().BoolP("dry-run", "n", false, "List repairs without changing files")
}

var includesFixCmd = &cobra.Command{
	Use:          "fix [NAMES...]",
	Short:        "Repair broken gateway include references",
	Long:         includeFixCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:               "true",
		cmd.CmdRequireHome:          "true",
		cmd.CmdWildcardNames:        "true",
		cmd.CmdNonInstanceArgsError: "true",
	},
	RunE: func(command *cobra.Command, _ []string) error {
		var (
			names  []string
			dryRun bool
			err    error
		)

		_, names, _, err = cmd.FetchArgs(command)
		if err != nil {
			return err
		}
		dryRun, err = command.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}
		return Fix(geneos.GetHost(cmd.Hostname), dryRun, names...)
	},
}

type includeFixSetup struct {
	gateway geneos.Instance
	setup   *geneosxml.Gateway
	path    string
}

func Fix(h *geneos.Host, dryRun bool, names ...string) error {
	if err := geneos.ValidateSite(config.Get[string](config.Global(), "site")); err != nil {
		return err
	}
	var (
		candidates = make(map[string][]string)
		seen       = make(map[string]bool)
		selected   = make(map[string]bool)
		root       = h.PathTo("gateway")
		setups     []includeFixSetup
		failures   []error
	)

	addCandidate := func(filename string) error {
		var (
			info     fs.FileInfo
			resolved string
			key      string
			err      error
		)

		info, err = h.Stat(filename)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		resolved, err = resolveIncludePath(h, filename, "")
		if err != nil {
			return err
		}
		key = h.Base(filename) + "\x00" + resolved
		if !seen[key] {
			seen[key] = true
			candidates[h.Base(filename)] = append(candidates[h.Base(filename)], filename)
		}
		return nil
	}
	for _, directory := range append([]string{"includes", "config", "gateway_config", geneos.SiteName()}, includeDirs...) {
		searchRoot := h.Join(root, directory)
		err := h.WalkDir(searchRoot, func(name string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if h.Ext(entry.Name()) != ".xml" {
				return nil
			}
			return addCandidate(h.Join(searchRoot, name))
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("scan include directory %q: %w", searchRoot, err)
		}
	}
	for _, i := range instance.Instances(h, nil, instance.MatchNames(names...)) {
		if !instance.IsA(i, "gateway") {
			continue
		}
		if err := instance.Read(i); err != nil {
			failures = append(failures, fmt.Errorf("gateway %q instance configuration: %w", i.Name(), err))
			continue
		}
		selected[instance.IDString(i)] = true
	}
	for _, i := range instance.Instances(h, geneos.ParseComponent("gateway")) {
		var (
			cf        = i.Config()
			setupPath = config.Get[string](cf, "setup")
			setup     *geneosxml.Gateway
			err       error
		)

		if !instance.IsA(i, "gateway") {
			continue
		}
		if remoteIncludeLocation(setupPath) {
			if selected[instance.IDString(i)] {
				fmt.Printf("Gateway %s: no changes made (HTTP(S) setup is not writable)\n", i.Name())
			}
			continue
		}
		if !h.IsAbs(setupPath) {
			setupPath = h.Join(i.Home(), setupPath)
		}
		setup, err = instance.ReadGatewaySetup(i)
		if err != nil {
			failures = append(failures, fmt.Errorf("gateway %q: %w", i.Name(), err))
			continue
		}
		setups = append(setups, includeFixSetup{gateway: i, setup: setup, path: setupPath})
		if setup.Includes != nil {
			walkFixIncludes(setup.Includes.Includes, setup.Includes.IncludeGroups, func(location *string) {
				filename := *location

				if *location == "" || remoteIncludeLocation(*location) {
					return
				}
				if !h.IsAbs(filename) {
					filename = h.Join(i.Home(), filename)
				}
				if err := addCandidate(filename); err != nil {
					failures = append(failures, fmt.Errorf("gateway %q include %q: %w", i.Name(), filename, err))
				}
			})
		}
	}
	for _, setup := range setups {
		if !selected[instance.IDString(setup.gateway)] {
			continue
		}
		if err := fixGatewayIncludes(h, setup, candidates, dryRun); err != nil {
			failures = append(failures, fmt.Errorf("gateway %q: %w", setup.gateway.Name(), err))
		}
	}
	return errors.Join(failures...)
}

func walkFixIncludes(includes []geneosxml.Include, groups []geneosxml.IncludeGroup, visit func(*string)) {
	for index := range includes {
		visit(&includes[index].Location)
	}
	for index := range groups {
		walkFixIncludes(groups[index].Includes, groups[index].IncludeGroups, visit)
	}
}

func fixGatewayIncludes(h *geneos.Host, setup includeFixSetup, candidates map[string][]string, dryRun bool) error {
	var (
		cf              = setup.gateway.Config()
		changes         = make(map[string]string)
		failures        []error
		includes        = config.Get[map[string]any](cf, "includes", config.NoExpand())
		updatedIncludes any
		configChanged   bool
		locations       []string
		modelCounts     = make(map[string]int)
	)

	if setup.setup.Includes == nil {
		fmt.Printf("Gateway %s: no changes made\n", setup.gateway.Name())
		return nil
	}
	walkFixIncludes(setup.setup.Includes.Includes, setup.setup.Includes.IncludeGroups, func(location *string) {
		filename := *location
		replacement, err := fixIncludeLocation(h, setup.gateway.Home(), *location, candidates)

		if err != nil {
			failures = append(failures, err)
			return
		}
		if replacement != *location {
			changes[*location] = replacement
			modelCounts[*location]++
			return
		}
		if *location != "" && !remoteIncludeLocation(*location) {
			if !h.IsAbs(filename) {
				filename = h.Join(setup.gateway.Home(), filename)
			}
			if info, err := h.Stat(filename); err != nil || !info.Mode().IsRegular() {
				fmt.Printf("Gateway %s: leaving %q unchanged (%d matching include files)\n", setup.gateway.Name(), *location, len(candidates[h.Base(*location)]))
			}
		}
	})
	if len(failures) > 0 {
		return errors.Join(failures...)
	}
	if len(changes) == 0 {
		fmt.Printf("Gateway %s: no changes made\n", setup.gateway.Name())
		return nil
	}

	// the setup is edited as text and not written from the model, so
	// that everything except the changed locations is left as it is
	var edited []byte
	if !dryRun {
		var err error
		if edited, err = editGatewaySetup(setup.gateway, changes, modelCounts); err != nil {
			return err
		}
	}
	updatedIncludes, configChanged = fixIncludeConfig(includes, changes)
	for location := range changes {
		locations = append(locations, location)
	}
	slices.Sort(locations)
	for _, location := range locations {
		fmt.Printf("Gateway %s: %s: %q -> %q\n", setup.gateway.Name(), setup.path, location, changes[location])
	}
	if configChanged {
		fmt.Printf("Gateway %s: update matching include references in the instance configuration\n", setup.gateway.Name())
	}
	if dryRun {
		return nil
	}
	backup, err := instance.WriteGatewaySetupBytes(setup.gateway, edited)
	auditIncludeFix(setup.gateway, setup.path, backup, changes, err)
	if err != nil {
		return err
	}
	if configChanged {
		configChanges := make(map[string]string)
		for _, location := range locations {
			if _, changed := fixIncludeConfig(includes, map[string]string{location: changes[location]}); changed {
				configChanges[location] = changes[location]
			}
		}
		config.Set(cf, "includes", updatedIncludes)
		err := instance.Write(setup.gateway, instance.NoRebuild(), instance.NoReload()).Err
		format := cf.ConfigType()
		if format == "" {
			format = instance.ConfigFileType()
		}
		auditIncludeFix(setup.gateway, instance.ComponentFilepath(setup.gateway, format), "", configChanges, err)
		return err
	}
	return nil
}

// editGatewaySetup returns the setup file of the gateway with the
// include locations in changes replaced. The model that the changes
// were found from and the file must agree on the number of locations
// that are replaced, otherwise nothing is written.
func editGatewaySetup(gateway geneos.Instance, changes map[string]string, modelCounts map[string]int) ([]byte, error) {
	raw, err := instance.ReadGatewaySetupBytes(gateway)
	if err != nil {
		return nil, fmt.Errorf("read setup: %w", err)
	}
	edited, counts, err := editIncludeLocations(raw, changes)
	if err != nil {
		return nil, fmt.Errorf("edit setup: %w", err)
	}
	for location, want := range modelCounts {
		if got := counts[location]; got != want {
			return nil, fmt.Errorf("model and raw setup file disagree: include %q changed %d times in the file but %d times in the parsed setup, no changes made", location, got, want)
		}
	}
	return edited, nil
}

func fixIncludeConfig(value any, changes map[string]string) (any, bool) {
	switch object := value.(type) {
	case string:
		prefix, location := "", object
		if separator := strings.LastIndex(object, "|"); separator >= 0 {
			prefix, location = object[:separator+1], strings.TrimSpace(object[separator+1:])
		}
		if replacement, ok := changes[location]; ok {
			if prefix != "" {
				return prefix + " " + replacement, true
			}
			return replacement, true
		}
	case map[string]any:
		updated := make(map[string]any, len(object))
		changed := false
		for key, item := range object {
			newValue, newChanged := fixIncludeConfig(item, changes)
			updated[key] = newValue
			changed = changed || newChanged
		}
		return updated, changed
	case []any:
		updated := make([]any, len(object))
		changed := false
		for index, item := range object {
			newValue, newChanged := fixIncludeConfig(item, changes)
			updated[index] = newValue
			changed = changed || newChanged
		}
		return updated, changed
	}
	return value, false
}

func remoteIncludeLocation(location string) bool {
	location = strings.ToLower(location)
	return strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://")
}

func fixIncludeLocation(h *geneos.Host, home, location string, candidates map[string][]string) (string, error) {
	var (
		filename    = location
		info        fs.FileInfo
		matches     = candidates[h.Base(location)]
		replacement string
		err         error
	)

	if location == "" || remoteIncludeLocation(location) {
		return location, nil
	}
	if !h.IsAbs(filename) {
		filename = h.Join(home, filename)
	}
	info, err = h.Stat(filename)
	if err == nil && info.Mode().IsRegular() {
		if replacement, err = siteConfigReplacement(h, filename); err != nil {
			return "", err
		}
		if replacement != "" {
			return includeReplacementLocation(h, home, replacement)
		}
		return location, nil
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("inspect include %q: %w", filename, err)
	}
	if len(matches) != 1 {
		return location, nil
	}
	replacement, err = h.Abs(matches[0])
	if err != nil {
		return "", err
	}
	return includeReplacementLocation(h, home, replacement)
}

// siteConfigReplacement returns the site configuration file that
// replaces an old config-directory or site-named file, but only when
// its contents match. Originals are retained.
func siteConfigReplacement(h *geneos.Host, filename string) (string, error) {
	legacy, err := h.Abs(h.PathTo("gateway", "includes", "config"))
	if err != nil {
		return "", err
	}
	abs, err := h.Abs(filename)
	if err != nil {
		return "", err
	}
	relative, err := h.Rel(legacy, abs)
	if err != nil {
		return "", nil
	}
	relative = h.ToSlash(relative)
	site := geneos.SiteName()
	oldSite := false
	if relative == ".." || strings.HasPrefix(relative, "../") {
		relative, err = h.Rel(h.PathTo("gateway", site), abs)
		if err != nil {
			return "", err
		}
		relative = h.ToSlash(relative)
		if relative == ".." || strings.HasPrefix(relative, "../") || !strings.HasPrefix(h.Base(relative), site+".") {
			return "", nil
		}
		oldSite = true
	}
	base := h.Base(relative)
	if oldSite {
		base = site + "-config" + strings.TrimPrefix(base, site)
	} else if strings.HasPrefix(base, "config.") {
		base = site + "-" + base
	}
	replacement, err := h.Abs(h.Join(h.PathTo("gateway", site), h.Dir(relative), base))
	if err != nil {
		return "", err
	}
	if replacement == abs {
		return "", nil
	}
	info, err := h.Stat(replacement)
	if errors.Is(err, fs.ErrNotExist) || err == nil && !info.Mode().IsRegular() {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect site config %q: %w", replacement, err)
	}
	same, err := sameFileContents(h, abs, replacement)
	if err != nil || !same {
		return "", err
	}
	return replacement, nil
}

func sameFileContents(h *geneos.Host, a, b string) (bool, error) {
	ia, err := h.Stat(a)
	if err != nil {
		return false, err
	}
	ib, err := h.Stat(b)
	if err != nil {
		return false, err
	}
	if ia.Size() != ib.Size() {
		return false, nil
	}
	fa, err := h.Open(a)
	if err != nil {
		return false, err
	}
	defer fa.Close()
	fb, err := h.Open(b)
	if err != nil {
		return false, err
	}
	defer fb.Close()
	return compareContents(fa, fb), nil
}

// includeReplacementLocation returns the location to use for the
// absolute path replacement in a gateway setup: relative to the gateway
// home if under the top-level `gateway` directory, otherwise absolute.
func includeReplacementLocation(h *geneos.Host, home, replacement string) (string, error) {
	root, err := h.Abs(h.PathTo("gateway"))
	if err != nil {
		return "", err
	}
	if replacement != root && strings.HasPrefix(h.ToSlash(replacement), strings.TrimSuffix(h.ToSlash(root), "/")+"/") {
		relative, err := h.Rel(home, replacement)
		if err != nil {
			return "", err
		}
		return h.ToSlash(relative), nil
	}
	return replacement, nil
}
