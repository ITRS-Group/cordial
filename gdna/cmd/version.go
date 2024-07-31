package cmd

import (
	_ "embed"
	"fmt"
	dbg "runtime/debug"

	"github.com/itrs-group/cordial"
	"github.com/spf13/cobra"
)

func init() {
	GDNACmd.AddCommand(versionCmd)
}

//go:embed _docs/version.md
var versionCmdDescription string

var versionCmd = &cobra.Command{
	Use:                   "version",
	Short:                 "Show program version",
	Long:                  versionCmdDescription,
	SilenceUsage:          true,
	Version:               cordial.VERSION,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s version %s\n", execname, cmd.Version)
		if debug {
			info, ok := dbg.ReadBuildInfo()
			if ok {
				fmt.Println("additional info:")
				fmt.Println(info)
			}
		}
	},
}
