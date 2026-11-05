package includes

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func updateIncludeLinks(h *geneos.Host, version string, override, missingOnly bool) error {
	ct := geneos.ParseComponent("gateway")
	dir := h.PathTo(ct, "includes", version)
	ents, err := h.ReadDir(dir)
	if err != nil {
		fmt.Println("Failed to read include directory", dir, ":", err)
		return err
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
	return nil
}

func includeVersionLink(h *geneos.Host, directory string) (string, error) {
	resolvedDirectory, err := resolveIncludePath(h, directory, "")
	if err != nil {
		return "", err
	}
	parent := h.Dir(directory)
	entries, err := h.ReadDir(parent)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		link := h.Join(parent, entry.Name())
		info, err := h.Lstat(link)
		if err != nil {
			return "", fmt.Errorf("failed to inspect include path %q: %w", link, err)
		}
		if info.Mode()&fs.ModeSymlink == 0 {
			continue
		}
		target, err := resolveIncludePath(h, link, resolvedDirectory)
		if err != nil {
			return "", fmt.Errorf("failed to resolve include link %q: %w", link, err)
		}
		if target == resolvedDirectory || strings.HasPrefix(target, resolvedDirectory+"/") {
			return link, nil
		}
	}
	return "", nil
}

func resolveIncludePath(h *geneos.Host, name, protectedDirectory string) (string, error) {
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
