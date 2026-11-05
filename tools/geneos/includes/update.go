package includes

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"

	goversion "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

//go:embed _docs/update.md
var includeUpdateCmdDescription string

var includesUpdateCmdVersion string

func init() {
	includesCmd.AddCommand(includesUpdateCmd)

	includesUpdateCmd.Flags().StringVarP(&includesUpdateCmdVersion, "version", "V", "", "Specify version to update includes to")
}

var includesUpdateCmd = &cobra.Command{
	Use:          "update",
	Args:         cobra.NoArgs,
	Short:        "Update includes",
	Long:         includeUpdateCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:       "false",
		cmd.CmdRequireHome:  "true",
		cmd.CmdAuditCommand: "never",
		cmd.CmdAuditActions: "never",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		h := geneos.GetHost(cmd.Hostname)

		err = Update(h, includesUpdateCmdVersion, true)
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
		version = cordial.VERSION
	}
	version = strings.TrimSpace(version)
	if _, err := goversion.NewVersion(version); err != nil {
		return fmt.Errorf("invalid include version %q: %w", version, err)
	}

	fmt.Println("Updating include file links to version", version)

	// first check if the desired version directory exists, and if it
	// does not and matches the current embedded version then extract
	dir := h.PathTo(ct, "includes", version)
	_, err = h.Stat(dir)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		if version == cordial.VERSION {
			log.Debug("Version directory does not exist, extracting includes for version", slog.String("version", version))
			err = Extract(h, override)
			if err != nil {
				log.Error("failed to extract includes", slog.String("version", version), slog.Any("error", err))
				return err
			}
		} else {
			return fmt.Errorf("directory not found for version %s", version)
		}
	}

	// now update the include file symlinks for the given version
	return updateIncludeLinks(h, version, override, false)
}
