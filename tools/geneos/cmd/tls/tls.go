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

package tls

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

func init() {
	cmd.RootCmd.AddCommand(TLSCmd)

	// tlsCmd.Flags().SortFlags = false
}

var TLSCmd = &cobra.Command{
	Use:   "tls",
	Short: "Manage certificates for secure connections",
	Long: strings.ReplaceAll(`
Manage certificates for [Geneos Secure Communications](https://docs.itrsgroup.com/docs/geneos/current/SSL/ssl_ug.html).

Sub-commands allow for initialisation, create and renewal of
certificates as well as listing details and copying a certificate
chain to all other hosts.
`, "|", "`"),
	SilenceUsage: true,
	Annotations:  make(map[string]string),
}
