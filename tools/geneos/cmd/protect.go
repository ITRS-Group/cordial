/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

var protectCmdUnprotect bool

func init() {
	GeneosCmd.AddCommand(protectCmd)

	protectCmd.Flags().BoolVarP(&protectCmdUnprotect, "unprotect", "U", false, "unprotect instances")
}

// protectCmd represents the protect command
var protectCmd = &cobra.Command{
	Use:     "protect [TYPE] [NAME...]",
	GroupID: GROUP_MANAGE,
	Short:   "Mark instances as protected",
	Long: strings.ReplaceAll(`
Mark matching instances as protected. Various operations that affect
the state or availability of an instance will be prevented if it is
marked |protected|.

To reverse this you must use the same command with the |-U| flag.
There is no |unprotect| command. This is by design.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args := CmdArgs(command)

		return instance.ForAll(ct, Hostname, protectInstance, args, []string{fmt.Sprintf("%v", !protectCmdUnprotect)})
	},
}

func protectInstance(c geneos.Instance, params []string) (err error) {
	cf := c.Config()

	var protect bool
	if len(params) > 0 {
		protect = params[0] == "true"
	}
	cf.Set("protected", protect)

	if cf.Type == "rc" {
		err = instance.Migrate(c)
	} else {
		err = cf.Save(c.Type().String(),
			config.Host(c.Host()),
			config.SaveDir(c.Type().InstancesDir(c.Host())),
			config.SetAppName(c.Name()),
		)
	}
	if err != nil {
		return
	}

	if protect {
		fmt.Printf("%s set to protected\n", c)
	} else {
		fmt.Printf("%s set to unprotected\n", c)
	}

	return
}
