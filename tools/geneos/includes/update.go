package includes

import (
	_ "embed"
	"errors"
	"io/fs"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

//go:embed _docs/update.md
var includeUpdateCmdDescription string

var includeUpdateCmdForce bool

func init() {
	includesCmd.AddCommand(includesUpdateCmd)

	includesUpdateCmd.Flags().BoolVarP(&includeUpdateCmdForce, "force", "F", false, "Overwrite existing files")
}

var includesUpdateCmd = &cobra.Command{
	Use:          "update",
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

		err = Update(h, "", includeUpdateCmdForce)
		if err != nil {
			return err
		}
		return nil
	},
}

// Update updates the include file symlinks for the given host,
// optionally overriding existing files. If version is not empty then us
// that version directory.
func Update(h *geneos.Host, version string, override bool) (err error) {
	ct := geneos.ParseComponent("gateway")
	if version == "" {
		version = cordial.VERSION
	}

	// first check if version directory exists, and if it does not and
	// is the current version then extract
	dir := h.PathTo(ct, "includes", version)
	_, err = h.Stat(dir)
	if err != nil && errors.Is(err, fs.ErrNotExist) && version == cordial.VERSION {
		err = Extract(h, override)
		if err != nil {
			log.Error("failed to extract includes", slog.String("version", version), slog.Any("error", err))
			return err
		}
	}

	// now update the include file symlinks for the given version
	ents, err := h.ReadDir(dir)
	if err != nil {
		log.Error("failed to read include directory", slog.String("dir", dir), slog.Any("error", err))
		return err
	}
	for _, ent := range ents {
		if ent.IsDir() {
			continue
		}

		newname := h.PathTo(ct, "includes", ent.Name())
		if err = h.Remove(newname); err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.Error("failed to remove old link", slog.String("link", newname), slog.Any("error", err))
			return err
		}
		oldname := h.Join(version, ent.Name())
		if err = h.Symlink(oldname, newname); err != nil {
			return err
		}
	}

	return nil
}
