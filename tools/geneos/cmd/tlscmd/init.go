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

package tlscmd

import (
	_ "embed"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/spf13/cobra"
)

var tlsInitCmdOverwrite bool

func init() {
	tlsCmd.AddCommand(tlsInitCmd)

	tlsInitCmd.Flags().BoolVarP(&tlsInitCmdOverwrite, "force", "F", false, "Overwrite any existing certificates")
	tlsInitCmd.Flags().SortFlags = false
}

//go:embed _docs/init.md
var tlsInitCmdDescription string

var tlsInitCmd = &cobra.Command{
	Use:                   "init",
	Short:                 "Initialise the TLS environment",
	Long:                  tlsInitCmdDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		// _, _, params := processArgsParams(cmd)
		return tlsInit()
	},
}

// create the tls/ directory in Geneos and a CA / DCA as required
//
// later options to allow import of a DCA
//
// This is also called from `init`
func tlsInit() (err error) {
	// directory permissions do not need to be restrictive
	err = geneos.LOCAL.MkdirAll(config.AppConfigDir(), 0775)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	if err := config.CreateRootCert(geneos.LOCAL, filepath.Join(config.AppConfigDir(), geneos.RootCAFile), tlsInitCmdOverwrite); err != nil {
		if errors.Is(err, geneos.ErrExists) {
			fmt.Println("root certificate already exists in", config.AppConfigDir())
			return nil
		}
	}
	fmt.Printf("CA certificate created for %s\n", geneos.RootCAFile)

	if err := config.CreateSigningCert(geneos.LOCAL, filepath.Join(config.AppConfigDir(), geneos.SigningCertFile), filepath.Join(config.AppConfigDir(), geneos.RootCAFile), tlsInitCmdOverwrite); err != nil {
		if errors.Is(err, geneos.ErrExists) {
			fmt.Println("signing certificate already exists in", config.AppConfigDir())
			return nil
		}
	}
	fmt.Printf("Signing certificate created for %s\n", geneos.SigningCertFile)

	return tlsSync()
}
