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
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

// aesLsCmd represents the aesLs command
var aesLsCmd = &cobra.Command{
	Use:                   "ls [TYPE] [NAME]",
	Short:                 "List configured AES key files",
	Long:                  `List configured AES key files`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return aesLSCommand(ct, args, params)
	},
}

func init() {
	aesCmd.AddCommand(aesLsCmd)
}

var aesLSTabWriter *tabwriter.Writer

func aesLSCommand(ct *geneos.Component, args []string, params []string) (err error) {
	aesLSTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
	fmt.Fprintf(aesLSTabWriter, "Type\tName\tHost\tKey-File\tCRC32\tModTime\n")
	err = instance.ForAll(ct, aesLSInstance, args, params)
	aesLSTabWriter.Flush()
	return
}

func aesLSInstance(c geneos.Instance, params []string) (err error) {
	path := instance.Filepath(c, "keyfile")
	if path == "" {
		return
	}
	s, err := c.Host().Stat(path)
	if err != nil {
		return
	}
	mtime := time.Unix(s.Mtime, 0)

	r, err := c.Host().Open(instance.Filepath(c, "keyfile"))
	if err != nil {
		return
	}
	defer r.Close()
	crc, err := config.Checksum(r)
	if err != nil {
		return
	}
	fmt.Fprintf(aesLSTabWriter, "%s\t%s\t%s\t%s\t%08X\t%s\n", c.Type(), c.Name(), c.Host(), path, crc, mtime.Format(time.RFC3339))
	return
}
