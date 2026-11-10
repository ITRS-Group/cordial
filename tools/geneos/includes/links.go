package includes

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"

	goversion "github.com/hashicorp/go-version"

	"github.com/itrs-group/cordial/pkg/config"
	geneosxml "github.com/itrs-group/cordial/pkg/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func updateIncludeLinks(h *geneos.Host, version string, override, missingOnly bool) error {
	ct := geneos.ParseComponent("gateway")
	dir := h.PathTo(ct, "includes", version)
	ents, err := h.ReadDir(dir)
	if err != nil {
		fmt.Println("Failed to read include directory", dir, ":", err)
		return err
	}
	var obsoleteLinks []string
	if !missingOnly {
		var replacements []string
		for _, ent := range ents {
			if !ent.IsDir() && filepath.Ext(ent.Name()) == ".xml" {
				replacements = append(replacements, ent.Name())
			}
		}
		obsoleteLinks, err = preflightIncludeLinks(h, version, replacements)
		if err != nil {
			return err
		}
	}
	for _, ent := range ents {
		if ent.IsDir() || filepath.Ext(ent.Name()) != ".xml" {
			continue
		}
		newname := h.PathTo(ct, "includes", ent.Name())
		oldname := h.Join(version, ent.Name())
		info, statErr := h.Lstat(newname)
		if statErr == nil {
			if missingOnly {
				continue
			}
			if info.Mode()&fs.ModeSymlink != 0 {
				linkTarget, readlinkErr := h.Readlink(newname)
				if readlinkErr == nil && linkTarget == oldname {
					fmt.Println("Symlink already up-to-date", newname, "->", oldname)
					continue
				}
			}
			if !override {
				return fmt.Errorf("include path %q already exists and replacement is disabled", newname)
			}
			if err = h.Remove(newname); err != nil {
				return fmt.Errorf("failed to remove existing include path %q: %w", newname, err)
			}
		} else if !errors.Is(statErr, fs.ErrNotExist) {
			return fmt.Errorf("failed to inspect include path %q: %w", newname, statErr)
		}
		if err = h.Symlink(oldname, newname); err != nil {
			return fmt.Errorf("failed to create symlink %q -> %q: %w", newname, oldname, err)
		}
		fmt.Println("Updated symlink", newname, "->", oldname)
	}
	return removeIncludeLinks(h, obsoleteLinks)
}

func includeVersionLink(h *geneos.Host, directory string) (string, error) {
	links, err := includeVersionLinks(h, directory)
	if err != nil || len(links) == 0 {
		return "", err
	}
	return links[0], nil
}

func includeVersionLinks(h *geneos.Host, directory string, replacements ...string) ([]string, error) {
	resolvedDirectory, err := resolveIncludePath(h, directory, "")
	if err != nil {
		return nil, err
	}
	parent := h.Dir(directory)
	entries, err := h.ReadDir(parent)
	if err != nil {
		return nil, err
	}
	var links []string
	for _, entry := range entries {
		if slices.Contains(replacements, entry.Name()) {
			continue
		}
		link := h.Join(parent, entry.Name())
		info, err := h.Lstat(link)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect include path %q: %w", link, err)
		}
		if info.Mode()&fs.ModeSymlink == 0 {
			continue
		}
		target, err := resolveIncludePath(h, link, resolvedDirectory)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve include link %q: %w", link, err)
		}
		if target == resolvedDirectory || strings.HasPrefix(target, resolvedDirectory+"/") {
			links = append(links, link)
		}
	}
	return links, nil
}

func unusedIncludeVersionLinks(h *geneos.Host, directory string, protectDirectory bool, replacements ...string) ([]string, string, error) {
	links, err := includeVersionLinks(h, directory, replacements...)
	if err != nil {
		return nil, "", err
	}
	if !protectDirectory && len(links) == 0 {
		return links, "", nil
	}
	resolvedDirectory, err := resolveIncludePath(h, directory, "")
	if err != nil {
		return nil, "", err
	}
	protectedDirectory := resolvedDirectory
	if !protectDirectory {
		protectedDirectory = ""
	}
	resolvedParent, err := resolveIncludePath(h, h.Dir(directory), "")
	if err != nil {
		return nil, "", err
	}
	protectedLinks := make([]string, len(links))
	for index, link := range links {
		protectedLinks[index] = path.Join(resolvedParent, h.ToSlash(h.Base(link)))
	}
	plannedLinks := slices.Clone(protectedLinks)
	for _, replacement := range replacements {
		plannedLinks = append(plannedLinks, path.Join(resolvedParent, replacement))
	}
	references := make(map[string][]string)
	for _, gw := range instance.Instances(h, geneos.ParseComponent("gateway")) {
		setup, err := instance.ReadGatewaySetup(gw)
		if err != nil {
			return nil, "", fmt.Errorf("failed to check includes for gateway %q: %w", gw.Name(), err)
		}
		locations := []string{config.Get[string](gw.Config(), "setup")}
		var addIncludes func([]geneosxml.Include, []geneosxml.IncludeGroup)
		addIncludes = func(includes []geneosxml.Include, groups []geneosxml.IncludeGroup) {
			for _, include := range includes {
				locations = append(locations, include.Location)
			}
			for _, group := range groups {
				addIncludes(group.Includes, group.IncludeGroups)
			}
		}
		if setup.Includes != nil {
			addIncludes(setup.Includes.Includes, setup.Includes.IncludeGroups)
		}
		for _, location := range locations {
			if strings.HasPrefix(location, "https://") || strings.HasPrefix(location, "http://") {
				continue
			}
			if !h.IsAbs(location) {
				location = h.Join(gw.Home(), location)
			}
			target, err := resolveIncludePath(h, location, protectedDirectory, plannedLinks...)
			if err != nil {
				return nil, "", fmt.Errorf("failed to resolve gateway %q include %q: %w", gw.Name(), location, err)
			}
			if slices.Contains(protectedLinks, target) || protectDirectory && (target == resolvedDirectory || strings.HasPrefix(target, resolvedDirectory+"/")) {
				if slices.Contains(protectedLinks, target) {
					location = target
				}
				references[location] = append(references[location], gw.Name())
			}
		}
	}
	if len(references) > 0 {
		var usage []string
		for location, gateways := range references {
			slices.Sort(gateways)
			gateways = slices.Compact(gateways)
			label := "gateway"
			if len(gateways) > 1 {
				label = "gateways"
			}
			usage = append(usage, fmt.Sprintf("%q (in use by %s %s)", location, label, strings.Join(gateways, ", ")))
		}
		slices.Sort(usage)
		return nil, strings.Join(usage, "; "), nil
	}
	return links, "", nil
}

func preflightIncludeLinks(h *geneos.Host, currentVersion string, replacements []string) ([]string, error) {
	parent := h.PathTo("gateway", "includes")
	entries, err := h.ReadDir(parent)
	if err != nil {
		return nil, err
	}
	var obsoleteLinks []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == currentVersion {
			continue
		}
		if _, err := goversion.NewVersion(entry.Name()); err != nil {
			continue
		}
		links, reference, err := unusedIncludeVersionLinks(h, h.Join(parent, entry.Name()), false, replacements...)
		if err != nil {
			return nil, err
		}
		if reference != "" {
			return nil, fmt.Errorf("%w: cannot update to version %q: obsolete links for version %q have no replacement: %s; update the gateway setups to remove or replace these include references, then retry; no include links were changed", ErrIncludeVersionLinked, currentVersion, entry.Name(), reference)
		}
		obsoleteLinks = append(obsoleteLinks, links...)
	}
	return obsoleteLinks, nil
}

func removeIncludeLinks(h *geneos.Host, links []string) error {
	for _, link := range links {
		if err := h.Remove(link); err != nil {
			return fmt.Errorf("failed to remove include link %q: %w", link, err)
		}
		fmt.Println("Removed unused include link", link)
	}
	return nil
}

func resolveIncludePath(h *geneos.Host, name, protectedDirectory string, protectedLinks ...string) (string, error) {
	absolute, err := h.Abs(name)
	if err != nil {
		return "", err
	}
	volume := h.ToSlash(h.VolumeName(absolute))
	root := volume + "/"
	resolved := root
	pending := strings.Split(strings.TrimPrefix(h.ToSlash(absolute), root), "/")
	links := 0
	for len(pending) > 0 {
		part := pending[0]
		pending = pending[1:]
		switch part {
		case "", ".":
			continue
		case "..":
			if resolved != root {
				resolved = path.Dir(resolved)
			}
			continue
		}
		candidate := path.Join(resolved, part)
		if slices.Contains(protectedLinks, candidate) {
			return candidate, nil
		}
		if protectedDirectory != "" && (candidate == protectedDirectory || strings.HasPrefix(candidate, protectedDirectory+"/")) {
			return candidate, nil
		}
		info, err := h.Lstat(candidate)
		if errors.Is(err, fs.ErrNotExist) {
			return path.Join(candidate, strings.Join(pending, "/")), nil
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&fs.ModeSymlink == 0 {
			resolved = candidate
			continue
		}
		links++
		if links > 40 {
			return "", fmt.Errorf("too many symbolic links resolving %q", name)
		}
		target, err := h.Readlink(candidate)
		if err != nil {
			return "", err
		}
		if h.IsAbs(target) {
			volume = h.ToSlash(h.VolumeName(target))
			root = volume + "/"
			resolved = root
			target = strings.TrimPrefix(h.ToSlash(target), root)
		} else {
			target = h.ToSlash(target)
		}
		pending = append(strings.Split(target, "/"), pending...)
	}
	return resolved, nil
}
