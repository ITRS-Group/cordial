package includes

import (
	_ "embed"
	"os"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

//go:embed _docs/install.md
var includeInstallCmdDescription string

var includeInstallCmdUpdate bool

func init() {
	includesCmd.AddCommand(includesInstallCmd)
	includesInstallCmd.Flags().BoolVarP(&includeInstallCmdUpdate, "update", "U", false, "Update version links for existing include files")
}

var includesInstallCmd = &cobra.Command{
	Use:          "install [ARCHIVE.tar.gz]",
	Aliases:      []string{"extract"},
	Args:         cobra.MaximumNArgs(1),
	Short:        "Install embedded includes or includes from an archive",
	Long:         includeInstallCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:       "false",
		cmd.CmdRequireHome:  "true",
		cmd.CmdAuditCommand: "never",
		cmd.CmdAuditActions: "never",
	},
	RunE: func(command *cobra.Command, args []string) error {
		var archive string
		if len(args) != 0 {
			archive = args[0]
		}
		return Install(geneos.GetHost(cmd.Hostname), archive, true, includeInstallCmdUpdate)
	},
}

func Install(h *geneos.Host, archive string, override, update bool) error {
	if archive == "" {
		if err := extractIncludes(h, IncludesFS, cordial.VERSION, override); err != nil {
			return err
		}
		return updateIncludeLinks(h, cordial.VERSION, override, !update)
	}
	source, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer source.Close()
	staging, err := os.MkdirTemp("", "geneos-includes-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	version, err := stageIncludeArchive(source, staging)
	if err != nil {
		return err
	}
	if err := extractIncludes(h, os.DirFS(staging), version, override); err != nil {
		return err
	}
	return updateIncludeLinks(h, version, override, !update)
}
