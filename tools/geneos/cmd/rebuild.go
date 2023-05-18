/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package cmd

import (
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var rebuildCmdForce, rebuildCmdReload bool

func init() {
	GeneosCmd.AddCommand(rebuildCmd)

	rebuildCmd.Flags().BoolVarP(&rebuildCmdForce, "force", "F", false, "Force rebuild")
	rebuildCmd.Flags().BoolVarP(&rebuildCmdReload, "reload", "r", false, "Reload instances after rebuild")
	rebuildCmd.Flags().SortFlags = false
}

var rebuildCmd = &cobra.Command{
	Use:     "rebuild [flags] [TYPE] [NAME...]",
	GroupID: GROUP_CONFIG,
	Short:   "Rebuild instance configuration files",
	Long: strings.ReplaceAll(`
All matching instances whose TYPE supported templates for
configuration file will have them rebuilt depending on the
|config.rebuild| setting for each instance.

The values for the |config.rebuild| option are: |never|, |initial|
and |always|. The default value depends on the TYPE; For Gateways it
is |initial| and for SANs and Floating Netprobes it is |always|.

You can force a rebuild for an instance that has the |config.rebuild|
set to |initial| by using the |--force|/|-F| option. Instances with a
|never| setting are never rebuilt.

The change this use something like |geneos set gateway
config.rebuild=always|

Instances will not normally update their settings when the
configuration file changes, although there are options for both
Gateways and Netprobes to do this, so you can trigger a configuration
reload with the |--reload|/|-r| option. This will send the
appropriate signal to matching instances regardless of the underlying
configuration being updated or not.

The templates use for each TYPE are stored in a |templates/|
directory under each TYPE. If you do not have templates because you
are adopting an existing installation or you have upgraded the
|geneos| program and want updated templates then run |geneos init
template| to overwrite existing file with the built-in ones.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := CmdArgsParams(cmd)
		return instance.ForAll(ct, Hostname, rebuildInstance, args, params)
	},
}

func rebuildInstance(c geneos.Instance, params []string) (err error) {
	if err = c.Rebuild(rebuildCmdForce); err != nil {
		return
	}
	log.Debug().Msgf("%s configuration rebuilt (if supported)", c)
	if !rebuildCmdReload {
		return
	}
	return reloadInstance(c, params)
}
