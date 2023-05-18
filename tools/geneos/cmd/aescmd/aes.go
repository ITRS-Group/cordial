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

// Package aescmd groups related AES256 keyfile and crypto commands
package aescmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/floating"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
)

var componentsWithKeyfiles = []*geneos.Component{&gateway.Gateway, &netprobe.Netprobe, &san.San, &floating.Floating}

func init() {
	cmd.GeneosCmd.AddCommand(AesCmd)
}

var AesCmd = &cobra.Command{
	Use:     "aes",
	GroupID: cmd.GROUP_SUBSYSTEMS,
	Short:   "Manage Geneos compatible key files and encode/decode passwords",
	Long: strings.ReplaceAll(`
Manage Geneos compatible key files and encode/decode passwords
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "true",
	},
	DisableFlagParsing:    true,
	DisableFlagsInUseLine: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		return cmd.RunE(command.Root(), []string{"aes", "ls"}, args)
	},
}
