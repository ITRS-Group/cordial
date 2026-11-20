package includes

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

//go:embed _docs/delete.md
var includeDeleteCmdDescription string

var ErrIncludeVersionLinked = errors.New("include version is referenced by a gateway")

func init() {
	includesCmd.AddCommand(includesDeleteCmd)
	includesDeleteCmd.Flags().StringP("version", "V", "", "Select include versions using a shell-style pattern")
	includesDeleteCmd.Flags().String("prefix", "itrs", "Select the include source prefix, or '*' for all sources")
}

var includesDeleteCmd = &cobra.Command{
	Use:     "delete [VERSION_PATTERN...]",
	Aliases: []string{"remove"},
	Args: func(command *cobra.Command, args []string) error {
		if len(args) > 0 && command.Flags().Changed("version") {
			return fmt.Errorf("specify version patterns either as arguments or with --version, not both")
		}
		return nil
	},
	Short:        "Delete unused include versions and links",
	Long:         includeDeleteCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "true",
	},
	RunE: func(command *cobra.Command, args []string) error {
		prefix, err := commandIncludePrefix(command, true)
		if err != nil {
			return err
		}
		pattern, err := command.Flags().GetString("version")
		if err != nil {
			return err
		}
		patterns := args
		if command.Flags().Changed("version") {
			patterns = []string{pattern}
		}
		for _, pattern := range patterns {
			if strings.TrimSpace(pattern) == "" {
				return fmt.Errorf("include version pattern must not be empty")
			}
		}
		return deleteIncludes(geneos.GetHost(cmd.Hostname), prefix, patterns...)
	},
}

func Delete(h *geneos.Host, patterns ...string) error {
	return deleteIncludes(h, "", patterns...)
}

func deleteIncludes(h *geneos.Host, prefix string, patterns ...string) error {
	if len(patterns) == 0 {
		return deleteIncludeVersionPattern(h, "", prefix)
	}
	if len(patterns) > 1 {
		for _, pattern := range patterns {
			if strings.TrimSpace(pattern) == "" {
				return fmt.Errorf("include version pattern must not be empty")
			}
		}
	}
	var failures []error
	for _, pattern := range patterns {
		if err := deleteIncludeVersionPattern(h, pattern, prefix); err != nil {
			failures = append(failures, fmt.Errorf("version pattern %q: %w", pattern, err))
		}
	}
	return errors.Join(failures...)
}

func deleteIncludeVersionPattern(h *geneos.Host, pattern, prefix string) error {
	pattern = strings.TrimSpace(pattern)
	allUnlinked := pattern == ""
	if allUnlinked {
		pattern = "*"
	}
	if strings.ContainsAny(pattern, `/\`) || pattern == "." || pattern == ".." {
		return fmt.Errorf("invalid include version pattern %q: directory paths are not allowed", pattern)
	}
	if _, err := path.Match(pattern, ""); err != nil {
		return fmt.Errorf("invalid include version pattern %q: %w", pattern, err)
	}
	parent := h.PathTo("gateway", "includes")
	entries, err := h.ReadDir(parent)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var directories []string
	var blocked []error
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		source, _, err := parseIncludeVersion(entry.Name())
		if err != nil || prefix != "" && source != prefix {
			continue
		}
		if matched, _ := path.Match(pattern, entry.Name()); !matched {
			continue
		}
		directory := h.Join(parent, entry.Name())
		info, err := h.Lstat(directory)
		if err != nil {
			return fmt.Errorf("failed to inspect include version %q: %w", entry.Name(), err)
		}
		if !info.IsDir() {
			continue
		}
		_, link, err := unusedIncludeVersionLinks(h, directory, true)
		if err != nil {
			return err
		}
		if link != "" {
			if allUnlinked {
				fmt.Println("Skipping referenced include version", entry.Name(), "referenced by", link)
			} else {
				blocked = append(blocked, fmt.Errorf("%w: cannot delete version %q, referenced by %s", ErrIncludeVersionLinked, entry.Name(), link))
			}
			continue
		}
		directories = append(directories, directory)
	}
	if len(blocked) > 0 {
		return fmt.Errorf("no selected versions were deleted: %w", errors.Join(blocked...))
	}
	for _, directory := range directories {
		info, err := h.Lstat(directory)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("include version path %q is no longer a directory", directory)
		}
		links, link, err := unusedIncludeVersionLinks(h, directory, true)
		if err != nil {
			return err
		}
		if link != "" {
			if allUnlinked {
				fmt.Println("Skipping referenced include version", h.Base(directory), "referenced by", link)
				continue
			}
			return fmt.Errorf("%w: cannot delete version %q, referenced by %s", ErrIncludeVersionLinked, h.Base(directory), link)
		}
		if err := removeIncludeLinks(h, links); err != nil {
			return err
		}
		if err := removeIncludeVersion(h, directory); err != nil {
			return fmt.Errorf("failed to delete include version %q: %w", h.Base(directory), err)
		}
		fmt.Println("Deleted include version", h.Base(directory))
	}
	return nil
}

func removeIncludeVersion(h *geneos.Host, directory string) error {
	var paths []string
	if err := h.WalkDir(directory, func(name string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		paths = append(paths, h.Join(directory, name))
		return nil
	}); err != nil {
		return fmt.Errorf("scan include version %q before deletion: %w", directory, err)
	}
	for _, path := range slices.Backward(paths) {
		err := h.Remove(path)
		auditIncludeFile(h, "delete", path, err)
		if err != nil {
			return err
		}
	}
	return nil
}
