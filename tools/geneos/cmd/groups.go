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

import "github.com/spf13/cobra"

// Available command groups for Cobra command set-up. This influences
// the display of the help text for the top-level `geneos` command.
const (
	CommandGroupConfig      = "config"
	CommandGroupComponents  = "components"
	CommandGroupCredentials = "credentials"
	CommandGroupManage      = "manage"
	CommandGroupProcess     = "process"
	CommandGroupSubsystems  = "subsystems"
	CommandGroupView        = "view"
)

func init() {
	GeneosCmd.AddGroup(&cobra.Group{
		ID:    CommandGroupSubsystems,
		Title: "Subsystems",
	})
	GeneosCmd.AddGroup(&cobra.Group{
		ID:    CommandGroupProcess,
		Title: "Control Instances",
	})
	GeneosCmd.AddGroup(&cobra.Group{
		ID:    CommandGroupView,
		Title: "Inspect Instances",
	})
	GeneosCmd.AddGroup(&cobra.Group{
		ID:    CommandGroupManage,
		Title: "Manage Instances",
	})
	GeneosCmd.AddGroup(&cobra.Group{
		ID:    CommandGroupConfig,
		Title: "Configure Instances",
	})
	GeneosCmd.AddGroup(&cobra.Group{
		ID:    CommandGroupCredentials,
		Title: "Manage Credentials",
	})
	GeneosCmd.AddGroup(&cobra.Group{
		ID:    CommandGroupComponents,
		Title: "Recognised Component Types",
	})
}
