package includes

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

//go:embed _docs/update.md
var includeUpdateCmdDescription string

var includesUpdateCmdVersion string

func init() {
	includesCmd.AddCommand(includesUpdateCmd)

	includesUpdateCmd.Flags().StringVarP(&includesUpdateCmdVersion, "version", "V", "", "Select a version or full source-version directory")
	includesUpdateCmd.Flags().String("prefix", "itrs", "Select the include source prefix")
}

var includesUpdateCmd = &cobra.Command{
	Use:          "update",
	Args:         cobra.NoArgs,
	Short:        "Update includes",
	Long:         includeUpdateCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "true",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		prefix, err := commandIncludePrefix(command, false)
		if err != nil {
			return err
		}
		version, err := includeVersionForPrefix(includesUpdateCmdVersion, prefix, command.Flags().Changed("prefix"))
		if err != nil {
			return err
		}
		h := geneos.GetHost(cmd.Hostname)
		if version == "" {
			version, err = latestIncludeVersion(h, prefix)
			if err != nil {
				return err
			}
		}

		err = Update(h, version, true)
		if err != nil {
			return err
		}
		return nil
	},
}

// Update updates the include file symlinks for the given host,
// optionally overriding existing files. If version is not empty then
// use that version directory.
func Update(h *geneos.Host, version string, override bool) (err error) {
	ct := geneos.ParseComponent("gateway")
	if version == "" {
		version = embeddedIncludeVersion()
	}
	version = strings.TrimSpace(version)
	if _, _, err := parseIncludeVersion(version); err != nil {
		return err
	}

	fmt.Println("Updating include file links to version", version)

	// first check if the desired version directory exists, and if it
	// does not and matches the current embedded version then extract
	dir := h.PathTo(ct, "includes", version)
	_, err = h.Stat(dir)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		if version == embeddedIncludeVersion() {
			log.Debug("Version directory does not exist, extracting includes for version", slog.String("version", version))
			err = Install(h, "", override, false)
			if err != nil {
				log.Error("failed to extract includes", slog.String("version", version), slog.Any("error", err))
				return err
			}
		} else {
			return fmt.Errorf("directory not found for version %s", version)
		}
	} else if err != nil {
		return err
	}

	// now update the include file symlinks for the given version
	return updateIncludeLinks(h, version, override, false)
}
