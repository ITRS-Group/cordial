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

// Package tlscmd contains all the TLS subsystem commands
package tlscmd

import (
	_ "embed"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

func init() {
	cmd.GeneosCmd.AddCommand(tlsCmd)
}

//go:embed README.md
var tlsCmdDescription string

var tlsCmd = &cobra.Command{
	Use:          "tls",
	GroupID:      cmd.CommandGroupSubsystems,
	Short:        "TLS Certificate Operations",
	Long:         tlsCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.AnnotationNeedsHome: "true",
	},
	DisableFlagParsing:    true,
	DisableFlagsInUseLine: true,
}
