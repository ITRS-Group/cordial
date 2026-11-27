package includes

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	includecontent "github.com/itrs-group/cordial/pkg/includes"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

//go:embed _docs/install.md
var includeInstallCmdDescription string

var includeInstallCmdUpdate bool

func init() {
	includesCmd.AddCommand(includesInstallCmd)
	includesInstallCmd.Flags().BoolVarP(&includeInstallCmdUpdate, "update", "U", false, "Update version links for existing include files")
	includesInstallCmd.Flags().String("prefix", "itrs", "Require this include source prefix (archives otherwise select their own source)")
}

var includesInstallCmd = &cobra.Command{
	Use:          "install [ARCHIVE.tar.gz]",
	Aliases:      []string{"extract"},
	Args:         cobra.MaximumNArgs(1),
	Short:        "Install embedded includes or includes from an archive",
	Long:         includeInstallCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "true",
	},
	RunE: func(command *cobra.Command, args []string) error {
		prefix, err := commandIncludePrefix(command, false)
		if err != nil {
			return err
		}
		var archive string
		if len(args) != 0 {
			archive = args[0]
		}
		if !command.Flags().Changed("prefix") {
			prefix = ""
		}
		return installIncludes(geneos.GetHost(cmd.Hostname), archive, true, includeInstallCmdUpdate, prefix)
	},
}

// Initialise fills missing bundled files and links without replacing existing
// includes, site configuration files, or links to other versions.
func Initialise(h *geneos.Host) error {
	version := embeddedIncludeVersion()
	if err := extractIncludeFiles(h, includecontent.Files, version, false, true); err != nil {
		return err
	}
	return updateIncludeLinks(h, version, false, true)
}

func Install(h *geneos.Host, archive string, override, update bool) error {
	return installIncludes(h, archive, override, update, "")
}

func installIncludes(h *geneos.Host, archive string, override, update bool, prefix string) error {
	if archive == "" {
		if err := checkIncludePrefix(embeddedIncludeVersion(), prefix, prefix != ""); err != nil {
			return fmt.Errorf("%w: only itrs includes are embedded; supply an archive for other sources", err)
		}
		if err := extractIncludes(h, includecontent.Files, embeddedIncludeVersion(), override); err != nil {
			return err
		}
		return updateIncludeLinks(h, embeddedIncludeVersion(), override, !update)
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
	if err := checkIncludePrefix(version, prefix, prefix != ""); err != nil {
		return err
	}
	if err := extractIncludes(h, os.DirFS(staging), version, override); err != nil {
		return err
	}
	return updateIncludeLinks(h, version, override, !update)
}
