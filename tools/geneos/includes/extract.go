package includes

import (
	_ "embed"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

//go:embed _docs/extract.md
var includeExtractCmdDescription string

var includeExtractCmdUpdate bool

func init() {
	includesCmd.AddCommand(includesExtractCmd)

	includesExtractCmd.Flags().BoolVarP(&includeExtractCmdUpdate, "update", "U", false, "Update version links for existing include files")
}

var includesExtractCmd = &cobra.Command{
	Use:          "extract",
	Args:         cobra.NoArgs,
	Short:        "Extract includes",
	Long:         includeExtractCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:       "false",
		cmd.CmdRequireHome:  "true",
		cmd.CmdAuditCommand: "never",
		cmd.CmdAuditActions: "never",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		h := geneos.GetHost(cmd.Hostname)

		err = Extract(h, true)
		if err != nil {
			return err
		}
		if includeExtractCmdUpdate {
			err = Update(h, "", true)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

// Extract extracts include files for the given host. If override is
// true, existing files will be overwritten. The files are placed in a
// version specific directory under the host's `gateway/includes` path.
// Missing top-level links are created without replacing existing paths.
func Extract(h *geneos.Host, override bool) (err error) {
	if err = extractIncludes(h, IncludesFS, cordial.VERSION, override); err != nil {
		return err
	}
	return updateIncludeLinks(h, cordial.VERSION, false, true)
}
