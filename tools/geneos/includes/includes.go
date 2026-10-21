package includes

import (
	"embed"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/logger"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

var log = logger.Logger

//go:embed *.xml
var IncludesFS embed.FS

//go:embed README.md
var includeCmdDescription string

func init() {
	cmd.Cmd.AddCommand(includesCmd)
}

var includesCmd = &cobra.Command{
	Use:          "includes",
	Aliases:      []string{"include"},
	GroupID:      cmd.CommandGroupSubsystems,
	Short:        "Manage includes",
	Long:         includeCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:       "false",
		cmd.CmdRequireHome:  "true",
		cmd.CmdAuditCommand: "never",
		cmd.CmdAuditActions: "never",
	},
	DisableFlagParsing:    true,
	DisableFlagsInUseLine: true,
}
