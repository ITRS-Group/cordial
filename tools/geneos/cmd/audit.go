package cmd

import (
	_ "embed"

	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(auditCmd)
}

//go:embed _docs/audit.md
var auditCmdDescription string

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit logging for Geneos commands",
	Long:  auditCmdDescription,
}
